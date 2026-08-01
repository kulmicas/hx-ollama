/*
  Copyright (c) 2009-2017 Dave Gamble and cJSON contributors
*/

#include <string.h>
#include <stdio.h>
#include <math.h>
#include <stdlib.h>
#include <limits.h>
#include <ctype.h>
#include "cJSON.h"

static cJSON *cJSON_New_Item(void) {
    cJSON* node = (cJSON*)calloc(1, sizeof(cJSON));
    return node;
}

void cJSON_Delete(cJSON *item) {
    cJSON *next = NULL;
    while (item) {
        next = item->next;
        if (item->child) { cJSON_Delete(item->child); }
        if (item->valuestring) { free(item->valuestring); }
        if (item->string) { free(item->string); }
        free(item);
        item = next;
    }
}

cJSON *cJSON_CreateObject(void) {
    cJSON *item = cJSON_New_Item();
    if (item) { item->type = cJSON_Object; }
    return item;
}

cJSON *cJSON_CreateString(const char *string) {
    cJSON *item = cJSON_New_Item();
    if (item) {
        item->type = cJSON_String;
        item->valuestring = strdup(string ? string : "");
    }
    return item;
}

cJSON *cJSON_CreateNumber(double num) {
    cJSON *item = cJSON_New_Item();
    if (item) {
        item->type = cJSON_Number;
        item->valuedouble = num;
        item->valueint = (int)num;
    }
    return item;
}

cJSON *cJSON_CreateBool(int boolean) {
    cJSON *item = cJSON_New_Item();
    if (item) { item->type = boolean ? cJSON_True : cJSON_False; }
    return item;
}

void cJSON_AddItemToObject(cJSON *object, const char *string, cJSON *item) {
    if (!object || !item) return;
    if (item->string) free(item->string);
    item->string = strdup(string);
    
    if (!object->child) {
        object->child = item;
    } else {
        cJSON *prev = object->child;
        while (prev->next) { prev = prev->next; }
        prev->next = item;
        item->prev = prev;
    }
}

cJSON *cJSON_AddStringToObject(cJSON * const object, const char * const name, const char * const string) {
    cJSON *item = cJSON_CreateString(string);
    cJSON_AddItemToObject(object, name, item);
    return item;
}

cJSON *cJSON_AddNumberToObject(cJSON * const object, const char * const name, const double number) {
    cJSON *item = cJSON_CreateNumber(number);
    cJSON_AddItemToObject(object, name, item);
    return item;
}

cJSON *cJSON_AddBoolToObject(cJSON * const object, const char * const name, const int boolean) {
    cJSON *item = cJSON_CreateBool(boolean);
    cJSON_AddItemToObject(object, name, item);
    return item;
}

cJSON *cJSON_GetObjectItemCaseSensitive(const cJSON * const object, const char * const string) {
    cJSON *current = NULL;
    if (!object || !string) return NULL;
    current = object->child;
    while (current) {
        if (current->string && strcmp(current->string, string) == 0) return current;
        current = current->next;
    }
    return NULL;
}

cJSON *cJSON_GetObjectItem(const cJSON * const object, const char * const string) {
    return cJSON_GetObjectItemCaseSensitive(object, string);
}

// Simple recursive descent parser
static const char *parse_value(cJSON *item, const char *value);

static const char *skip_whitespace(const char *in) {
    while (in && *in && (unsigned char)*in <= 32) in++;
    return in;
}

static const char *parse_string(cJSON *item, const char *str) {
    const char *ptr = str + 1;
    char *ptr2;
    char *out;
    int len = 0;
    if (*str != '\"') return NULL;
    
    while (*ptr != '\"' && *ptr && ++len) {
        if (*ptr++ == '\\') ptr++;
    }
    
    out = (char*)malloc(len + 1);
    if (!out) return NULL;
    
    ptr = str + 1;
    ptr2 = out;
    while (*ptr != '\"' && *ptr) {
        if (*ptr != '\\') *ptr2++ = *ptr++;
        else {
            ptr++;
            switch (*ptr) {
                case 'b': *ptr2++ = '\b'; break;
                case 'f': *ptr2++ = '\f'; break;
                case 'n': *ptr2++ = '\n'; break;
                case 'r': *ptr2++ = '\r'; break;
                case 't': *ptr2++ = '\t'; break;
                case '\"': *ptr2++ = '\"'; break;
                case '\\': *ptr2++ = '\\'; break;
                case '/': *ptr2++ = '/'; break;
                default: *ptr2++ = *ptr; break;
            }
            ptr++;
        }
    }
    *ptr2 = 0;
    if (*ptr == '\"') ptr++;
    item->type = cJSON_String;
    item->valuestring = out;
    return ptr;
}

static const char *parse_number(cJSON *item, const char *num) {
    double n = 0;
    char *endptr;
    n = strtod(num, &endptr);
    item->valuedouble = n;
    item->valueint = (int)n;
    item->type = cJSON_Number;
    return endptr;
}

static const char *parse_array(cJSON *item, const char *value) {
    cJSON *child;
    if (*value != '[') return NULL;
    item->type = cJSON_Array;
    value = skip_whitespace(value + 1);
    if (*value == ']') return value + 1;

    item->child = child = cJSON_New_Item();
    value = skip_whitespace(parse_value(child, value));
    if (!value) return NULL;

    while (*value == ',') {
        cJSON *new_item = cJSON_New_Item();
        child->next = new_item;
        new_item->prev = child;
        child = new_item;
        value = skip_whitespace(parse_value(child, value + 1));
        if (!value) return NULL;
    }

    if (*value == ']') return value + 1;
    return NULL;
}

static const char *parse_object(cJSON *item, const char *value) {
    cJSON *child;
    if (*value != '{') return NULL;
    item->type = cJSON_Object;
    value = skip_whitespace(value + 1);
    if (*value == '}') return value + 1;

    item->child = child = cJSON_New_Item();
    value = skip_whitespace(parse_string(child, value));
    if (!value) return NULL;
    child->string = child->valuestring;
    child->valuestring = NULL;

    if (*value != ':') return NULL;
    value = skip_whitespace(parse_value(child, skip_whitespace(value + 1)));
    if (!value) return NULL;

    while (*value == ',') {
        cJSON *new_item = cJSON_New_Item();
        child->next = new_item;
        new_item->prev = child;
        child = new_item;
        value = skip_whitespace(parse_string(child, skip_whitespace(value + 1)));
        if (!value) return NULL;
        child->string = child->valuestring;
        child->valuestring = NULL;
        if (*value != ':') return NULL;
        value = skip_whitespace(parse_value(child, skip_whitespace(value + 1)));
        if (!value) return NULL;
    }

    if (*value == '}') return value + 1;
    return NULL;
}

static const char *parse_value(cJSON *item, const char *value) {
    if (!value) return NULL;
    if (!strncmp(value, "null", 4)) { item->type = cJSON_NULL; return value + 4; }
    if (!strncmp(value, "false", 5)) { item->type = cJSON_False; return value + 5; }
    if (!strncmp(value, "true", 4)) { item->type = cJSON_True; return value + 4; }
    if (*value == '\"') { return parse_string(item, value); }
    if (*value == '-' || (*value >= '0' && *value <= '9')) { return parse_number(item, value); }
    if (*value == '[') { return parse_array(item, value); }
    if (*value == '{') { return parse_object(item, value); }
    return NULL;
}

cJSON *cJSON_Parse(const char *value) {
    cJSON *c = cJSON_New_Item();
    if (!c) return NULL;
    if (!parse_value(c, skip_whitespace(value))) {
        cJSON_Delete(c);
        return NULL;
    }
    return c;
}

// Print unformatted JSON string generator
static char *print_value(const cJSON *item) {
    if (!item) return strdup("");
    if (item->type == cJSON_String) {
        char *buf = (char*)malloc(strlen(item->valuestring) * 2 + 3);
        sprintf(buf, "\"%s\"", item->valuestring);
        return buf;
    }
    if (item->type == cJSON_Number) {
        char *buf = (char*)malloc(64);
        sprintf(buf, "%g", item->valuedouble);
        return buf;
    }
    if (item->type == cJSON_True) return strdup("true");
    if (item->type == cJSON_False) return strdup("false");
    if (item->type == cJSON_NULL) return strdup("null");
    
    if (item->type == cJSON_Object) {
        char *out = strdup("{");
        cJSON *child = item->child;
        while (child) {
            char *val = print_value(child);
            char *tmp = (char*)malloc(strlen(out) + strlen(child->string) + strlen(val) + 10);
            sprintf(tmp, "%s\"%s\":%s%s", out, child->string, val, child->next ? "," : "");
            free(out); free(val);
            out = tmp;
            child = child->next;
        }
        char *tmp = (char*)malloc(strlen(out) + 2);
        sprintf(tmp, "%s}", out);
        free(out);
        return tmp;
    }
    return strdup("{}");
}

char *cJSON_PrintUnformatted(const cJSON *item) {
    return print_value(item);
}
