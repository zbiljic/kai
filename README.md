# kai

[![Go Report Card](https://goreportcard.com/badge/github.com/zbiljic/kai)](https://goreportcard.com/report/github.com/zbiljic/kai)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`kai` is a command-line interface (CLI) tool that leverages Artificial Intelligence to automatically generate Git commit messages based on your staged changes. It aims to streamline your commit workflow by providing concise, relevant, and optionally Conventional Commit-formatted messages, allowing you to focus more on coding and less on crafting perfect commit messages.

## ✨ Features

*   **AI-Powered Generation**: Automatically creates commit messages by analyzing the diff of your staged Git changes.
*   **Conventional Commits Support**: Generates messages adhering to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification (e.g., `feat(scope): add new feature`).
*   **Interactive Workflow**: Provides a selection of generated messages and allows interactive editing before committing.
*   **Multiple LLM Providers**: Supports various Large Language Model providers for flexibility:
    *   [Phind](https://www.phind.com/) (Default)
    *   [OpenAI](https://openai.com/) (GPT-4o Mini, GPT-3.5 Turbo, etc.)
    *   [Google AI](https://ai.google.dev/) (Gemini models)
    *   [OpenRouter](https://openrouter.ai/) (various models, including MistralAI)
*   **Go-powered**: Built with Go, offering a single, fast binary.

## 🚀 Installation

`kai` is a Go application. You'll need `Go 1.23+` installed on your system.

### Using `go install`

The easiest way to install `kai` is using `go install`:

```bash
go install github.com/zbiljic/kai@latest
```

Ensure that your `GOBIN` is in your system's `PATH` to run `kai` directly from your terminal.

### Building from source

Alternatively, you can clone the repository and build `kai` yourself:

```bash
git clone https://github.com/zbiljic/kai.git
cd kai
make install
```

## 💡 Usage

`kai` is designed to be used within a Git repository with staged changes.

1.  **Stage your changes**: Before running `kai`, make sure you have files staged for commit:
    ```bash
    git add .
    ```

2.  **Generate a commit message**:
    Simply run `kai` or `kai gen` in your repository:
    ```bash
    kai
    # or
    kai gen
    ```
    `kai` will analyze your staged changes, generate potential commit messages, and present them in an interactive prompt:

    ```
    ◆ Pick a commit message to use: (Ctrl+c to exit)
      ● [1] feat: add new feature (e to edit)
      ○ [2] fix: resolve bug
      ○ [3] chore: update dependencies
    ```

    You can navigate through the suggestions using arrow keys. Press `Enter` to select a message or `e` to edit the selected message interactively.

    It is also possible to quickly pick a message by typing number in front of it.

    If you choose to edit, `kai` will guide you through modifying the type, scope, and message body, especially useful for adhering to Conventional Commits.

3.  **Commit**: Once you select or confirm a message, `kai` will automatically commit your staged changes with the chosen message.

### Options

*   **Specify LLM Provider**: Use the `--provider` or `-p` flag to choose your desired LLM provider.
    ```bash
    kai gen --provider openai
    kai gen -p googleai
    ```
    Available providers: `phind` (default), `openai`, `googleai`, `openrouter`.

*   **Specify Commit Message Type**: Use the `--type` or `-t` flag to set the desired commit message format.
    ```bash
    kai gen --type simple
    kai gen -t conventional
    ```
    Available types: `conventional` (default), `simple`.
    *   `conventional`: Generates messages like `type(scope): message`.
    *   `simple`: Generates plain messages like `message`.

## ⚙️ Configuration

`kai` relies on environment variables for API keys to access LLM providers.

*   **OpenAI**: Set your API key for OpenAI:
    ```bash
    export OPENAI_API_KEY="your_openai_api_key"
    ```

*   **Google AI**: Set your API key for Google Gemini:
    ```bash
    export GEMINI_API_KEY="your_google_ai_api_key"
    ```

*   **OpenRouter**: Set your API key for OpenRouter:
    ```bash
    export OPENROUTER_API_KEY="your_openrouter_api_key"
    ```
    With OpenRouter it is also possible setting these for custom attribution:
    ```bash
    export OPENROUTER_HTTP_REFERER="https://github.com/zbiljic/kai"
    export OPENROUTER_X_TITLE="kai"
    ```

## 🤝 Contributing

Contributions are welcome! If you find a bug or have a feature request, please open an issue or submit a pull request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

*   Inspired by other AI-powered commit tools and the need for a simple Go solution that allows easy message modification.
*   Uses [go-clack](https://github.com/Mist3rBru/go-clack) for interactive prompts.

## Similar Projects

Here are some other similar projects that you might find useful:

*   [aicommits](https://github.com/Nutlope/aicommits): A CLI that writes your Git commit messages for you with AI. (TypeScript)
*   [lumen](https://github.com/jnsahaj/lumen): Instant AI Git Commit message, Git changes summary from the CLI. (Rust)
