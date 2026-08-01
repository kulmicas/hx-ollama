/*
  hx-ollama.c: Fast, Zero-Dependency C Static Binary for Helix Editor + Ollama AI
  Supports POSIX Sockets, local or LAN Ollama hosts (OLLAMA_HOST), ~/.config/hx-ollama/config.json,
  and STB-style cJSON parsing.
*/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <netdb.h>
#include <arpa/inet.h>
#include <sys/stat.h>
#include <ctype.h>
#include "cJSON.h"

#define DEFAULT_HOST "http://localhost:11434"
#define DEFAULT_MODEL "qwen2.5-coder:14b-instruct"

// System Prompts
static const char *SYSTEM_PROMPT_EDIT = 
    "You are an expert AI coding assistant integrated into the Helix text editor.\n"
    "Your task is to edit, refactor, or rewrite the provided code based on the user's instructions.\n"
    "CRITICAL RULE: Output ONLY the updated code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.\n"
    "Do NOT include any introduction, explanations, markdown formatting, or conversational text.\n"
    "Your entire response will replace the user's selection in the editor.";

static const char *SYSTEM_PROMPT_FIX = 
    "You are an expert AI debugger integrated into the Helix text editor.\n"
    "Your task is to analyze the provided code snippet, identify any syntax errors, logical bugs, or type mismatches, and fix them.\n"
    "CRITICAL RULE: Output ONLY the corrected code. Do NOT wrap your output in markdown code blocks or ``` ``` fences.\n"
    "Do NOT include any introduction, explanations, or conversational text.\n"
    "Your entire response will replace the user's selection in the editor.";

static const char *SYSTEM_PROMPT_EXPLAIN = 
    "You are an expert software developer and technical communicator integrated into Helix text editor.\n"
    "Analyze the provided code selection and explain clearly how it works, key data structures, algorithms, and potential edge cases.\n"
    "Format your output with clear, concise markdown headings and bullet points.";

static const char *SYSTEM_PROMPT_DOCS = 
    "You are an expert AI code documenter integrated into Helix text editor.\n"
    "Add clear, concise docstrings, inline comments, and type hints/annotations to the provided code following standard style guidelines for the language.\n"
    "CRITICAL RULE: Output ONLY the code with documentation added. Do NOT wrap your output in markdown code blocks or ``` ``` fences.";

static const char *SYSTEM_PROMPT_GENERATE = 
    "You are an expert AI software developer integrated into Helix text editor.\n"
    "Generate clean, production-ready code based on the user's prompt instruction.\n"
    "CRITICAL RULE: Output ONLY the generated code unless explicitly asked for explanation. Do NOT wrap your output in markdown code blocks or ``` ``` fences unless requested.";

static const char *HELIX_CONFIG_SNIPPET = 
    "\n# ==============================================================================\n"
    "# Helix Editor + Ollama AI Integration (hx-ollama)\n"
    "# ==============================================================================\n\n"
    "[keys.normal.space.o]\n"
    "g = \"@:append-output<space>hx-ollama<space>generate<space>\"\n"
    "i = \"@:insert-output<space>hx-ollama<space>generate<space>\"\n"
    "m = \":sh hx-ollama models\"\n\n"
    "[keys.select.space.o]\n"
    "e = \"@|hx-ollama edit<space>\"\n"
    "f = \":pipe hx-ollama fix\"\n"
    "x = \":pipe hx-ollama explain\"\n"
    "d = \":pipe hx-ollama docs\"\n"
    "c = \":pipe hx-ollama complete\"\n";

// Helper: Strip code fences ```lang ... ```
static char *strip_code_fences(const char *text) {
    if (!text || !*text) return strdup("");
    
    char *out = strdup(text);
    char *p = out;
    
    while (isspace((unsigned char)*p)) p++;
    
    if (strncmp(p, "```", 3) == 0) {
        char *first_line_end = strchr(p, '\n');
        if (first_line_end) {
            p = first_line_end + 1;
            char *last_fence = strrchr(p, '`');
            if (last_fence && last_fence >= p + 2 && strncmp(last_fence - 2, "```", 3) == 0) {
                *(last_fence - 2) = '\0';
            }
            char *res = strdup(p);
            free(out);
            return res;
        }
    }
    return out;
}

// Helper: Non-blocking stdin read
static char *read_stdin_nonblocking(void) {
    fd_set readfds;
    struct timeval tv = {0, 50000}; // 50ms
    FD_ZERO(&readfds);
    FD_SET(STDIN_FILENO, &readfds);

    if (select(STDIN_FILENO + 1, &readfds, NULL, NULL, &tv) > 0) {
        size_t cap = 4096, len = 0;
        char *buf = malloc(cap);
        char tmp[1024];
        ssize_t n;
        while ((n = read(STDIN_FILENO, tmp, sizeof(tmp))) > 0) {
            if (len + n >= cap) {
                cap *= 2;
                buf = realloc(buf, cap);
            }
            memcpy(buf + len, tmp, n);
            len += n;
        }
        buf[len] = '\0';
        return buf;
    }
    return NULL;
}

// Config file reader (~/.config/hx-ollama/config.json)
static void load_config_file(char *host_out, size_t host_size, char *model_out, size_t model_size) {
    const char *home = getenv("HOME");
    if (!home) return;
    
    char cfg_path[512];
    snprintf(cfg_path, sizeof(cfg_path), "%s/.config/hx-ollama/config.json", home);
    
    FILE *f = fopen(cfg_path, "r");
    if (!f) return;
    
    fseek(f, 0, SEEK_END);
    long sz = ftell(f);
    fseek(f, 0, SEEK_SET);
    
    if (sz > 0) {
        char *buf = malloc(sz + 1);
        fread(buf, 1, sz, f);
        buf[sz] = '\0';
        
        cJSON *json = cJSON_Parse(buf);
        if (json) {
            cJSON *h = cJSON_GetObjectItem(json, "host");
            if (h && h->valuestring && *h->valuestring) {
                strncpy(host_out, h->valuestring, host_size - 1);
            }
            cJSON *m = cJSON_GetObjectItem(json, "model");
            if (m && m->valuestring && *m->valuestring) {
                strncpy(model_out, m->valuestring, model_size - 1);
            }
            cJSON_Delete(json);
        }
        free(buf);
    }
    fclose(f);
}

// Socket HTTP Request (GET or POST)
static char *http_request(const char *method, const char *host_url, const char *path, const char *json_payload) {
    char hostname[256] = "localhost";
    int port = 11434;

    const char *p = host_url;
    if (strncmp(p, "http://", 7) == 0) p += 7;
    else if (strncmp(p, "https://", 8) == 0) p += 8;

    char *colon = strchr(p, ':');
    if (colon) {
        size_t hlen = colon - p;
        if (hlen < sizeof(hostname)) {
            strncpy(hostname, p, hlen);
            hostname[hlen] = '\0';
        }
        port = atoi(colon + 1);
    } else {
        strncpy(hostname, p, sizeof(hostname) - 1);
    }

    struct hostent *server = gethostbyname(hostname);
    if (!server) {
        fprintf(stderr, "[hx-ollama] Error: Unknown host '%s'\n", hostname);
        return NULL;
    }

    int sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd < 0) {
        perror("[hx-ollama] Error opening socket");
        return NULL;
    }

    struct sockaddr_in serv_addr;
    memset(&serv_addr, 0, sizeof(serv_addr));
    serv_addr.sin_family = AF_INET;
    memcpy(&serv_addr.sin_addr.s_addr, server->h_addr, server->h_length);
    serv_addr.sin_port = htons(port);

    if (connect(sockfd, (struct sockaddr *)&serv_addr, sizeof(serv_addr)) < 0) {
        fprintf(stderr, "[hx-ollama] Error: Could not connect to Ollama at %s:%d\n", hostname, port);
        close(sockfd);
        return NULL;
    }

    char header[1024];
    size_t payload_len = json_payload ? strlen(json_payload) : 0;
    int header_len = snprintf(header, sizeof(header),
        "%s %s HTTP/1.1\r\n"
        "Host: %s:%d\r\n"
        "Content-Type: application/json\r\n"
        "Content-Length: %zu\r\n"
        "Connection: close\r\n\r\n",
        method, path, hostname, port, payload_len);

    write(sockfd, header, header_len);
    if (json_payload && payload_len > 0) {
        write(sockfd, json_payload, payload_len);
    }

    size_t resp_cap = 16384, resp_len = 0;
    char *response = malloc(resp_cap);
    ssize_t bytes_read;
    char buffer[4096];

    while ((bytes_read = read(sockfd, buffer, sizeof(buffer))) > 0) {
        if (resp_len + bytes_read >= resp_cap) {
            resp_cap *= 2;
            response = realloc(response, resp_cap);
        }
        memcpy(response + resp_len, buffer, bytes_read);
        resp_len += bytes_read;
    }
    response[resp_len] = '\0';
    close(sockfd);

    char *body = strstr(response, "\r\n\r\n");
    if (body) {
        char *ret = strdup(body + 4);
        free(response);
        return ret;
    }

    return response;
}

static char *format_output(const char *raw, int code_only) {
    if (code_only) {
        return strip_code_fences(raw);
    }
    return strdup(raw);
}

int main(int argc, char **argv) {
    const char *action = "";
    const char *custom_prompt = "";
    
    char host_buf[256] = DEFAULT_HOST;
    char model_buf[128] = DEFAULT_MODEL;
    
    load_config_file(host_buf, sizeof(host_buf), model_buf, sizeof(model_buf));

    if (getenv("OLLAMA_HOST")) {
        strncpy(host_buf, getenv("OLLAMA_HOST"), sizeof(host_buf) - 1);
    }

    const char *host = host_buf;
    const char *model = model_buf;
    int keep_code = 0;
    int code_only = 1;

    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--host") == 0 && i + 1 < argc) host = argv[++i];
        else if (strcmp(argv[i], "-m") == 0 && i + 1 < argc) model = argv[++i];
        else if (strcmp(argv[i], "--raw") == 0) code_only = 1;
        else if (strcmp(argv[i], "--markdown") == 0) code_only = 0;
        else if (strcmp(argv[i], "--keep-code") == 0) keep_code = 1;
        else if (!*action) action = argv[i];
        else custom_prompt = argv[i];
    }

    if (strcmp(action, "setup") == 0 || strcmp(action, "init") == 0 || strcmp(action, "install-helix") == 0) {
        printf("=================================================================\n");
        printf("   hx-ollama C Static Binary Location Overview\n");
        printf("=================================================================\n");
        printf("1. Target Binary: ~/.local/bin/hx-ollama\n");
        printf("2. Config File:   ~/.config/hx-ollama/config.json\n");
        printf("3. Helix Config:  ~/.config/helix/config.toml\n");
        printf("=================================================================\n\n");
        printf("Helix Configuration Snippet:\n%s\n", HELIX_CONFIG_SNIPPET);
        return 0;
    }

    if (strcmp(action, "models") == 0) {
        char *resp = http_request("GET", host, "/api/tags", NULL);
        if (resp) {
            cJSON *json = cJSON_Parse(resp);
            if (json) {
                cJSON *models = cJSON_GetObjectItem(json, "models");
                if (models && models->type == cJSON_Array) {
                    printf("Installed Models on %s:\n", host);
                    cJSON *m = models->child;
                    while (m) {
                        cJSON *name = cJSON_GetObjectItem(m, "name");
                        if (name && name->valuestring) {
                            printf("  - %s\n", name->valuestring);
                        }
                        m = m->next;
                    }
                }
                cJSON_Delete(json);
            }
            free(resp);
        }
        return 0;
    }

    char *stdin_text = read_stdin_nonblocking();
    const char *sys_prompt = SYSTEM_PROMPT_EDIT;

    if (strcmp(action, "fix") == 0) sys_prompt = SYSTEM_PROMPT_FIX;
    else if (strcmp(action, "explain") == 0) {
        sys_prompt = SYSTEM_PROMPT_EXPLAIN;
        code_only = 0;
        keep_code = 1;
    } else if (strcmp(action, "docs") == 0) sys_prompt = SYSTEM_PROMPT_DOCS;
    else if (strcmp(action, "generate") == 0) sys_prompt = SYSTEM_PROMPT_GENERATE;

    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "model", model);
    cJSON_AddBoolToObject(root, "stream", 0);
    cJSON_AddStringToObject(root, "system", sys_prompt);

    char full_prompt[8192];
    if (stdin_text && *stdin_text) {
        snprintf(full_prompt, sizeof(full_prompt), "User Request: %s\n\nCode Context:\n%s", 
            *custom_prompt ? custom_prompt : action, stdin_text);
    } else {
        snprintf(full_prompt, sizeof(full_prompt), "%s %s", action, custom_prompt);
    }

    cJSON_AddStringToObject(root, "prompt", full_prompt);

    cJSON *options = cJSON_CreateObject();
    cJSON_AddNumberToObject(options, "temperature", 0.2);
    cJSON_AddItemToObject(root, "options", options);

    char *payload = cJSON_PrintUnformatted(root);
    cJSON_Delete(root);

    char *resp_body = http_request("POST", host, "/api/generate", payload);
    free(payload);

    if (resp_body) {
        cJSON *resp_json = cJSON_Parse(resp_body);
        if (resp_json) {
            cJSON *response_item = cJSON_GetObjectItem(resp_json, "response");
            if (response_item && response_item->valuestring) {
                char *formatted = format_output(response_item->valuestring, code_only);
                if (keep_code && stdin_text && *stdin_text) {
                    printf("%s\n\n---\n### 💡 Code Explanation\n%s\n", stdin_text, formatted);
                } else {
                    printf("%s", formatted);
                }
                free(formatted);
            }
            cJSON_Delete(resp_json);
        }
        free(resp_body);
    }

    if (stdin_text) free(stdin_text);
    return 0;
}
