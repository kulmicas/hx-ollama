# Contributing to `hx-ollama`

Thank you for considering contributing to `hx-ollama`! We welcome bug reports, feature requests, documentation improvements, and pull requests.

---

## 🛠️ Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-username/hx-ollama.git
   cd hx-ollama
   ```

2. **Build locally**:
   ```bash
   make
   ```

3. **Test the binary**:
   ```bash
   ./bin/hx-ollama --help
   ./bin/hx-ollama models
   ```

---

## 💡 Code Style Guidelines

- Follow standard Go formatting (`gofmt`).
- Maintain zero external third-party dependencies (`net/http`, `encoding/json`, `os`, `flag`).
- Ensure fail-safe protection: functions handling editor selections MUST echo original `stdin` back to `stdout` on network or API failures.

---

## 📄 License

By contributing, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
