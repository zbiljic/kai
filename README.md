# kai

[![Go Report Card](https://goreportcard.com/badge/github.com/zbiljic/kai)](https://goreportcard.com/report/github.com/zbiljic/kai)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`kai` is a command-line interface (CLI) tool that leverages Artificial Intelligence to automatically generate Git commit messages based on your staged changes. It aims to streamline your commit workflow by providing concise, relevant, and optionally Conventional Commit-formatted messages, allowing you to focus more on coding and less on crafting perfect commit messages.

## ✨ Features

*   **AI-Powered Generation**: Automatically creates commit messages by analyzing the diff of your staged Git changes.
*   **Conventional Commits Support**: Generates messages adhering to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification (e.g., `feat(scope): add new feature`).
*   **Interactive Workflow**: Provides a selection of generated messages and allows interactive editing before committing. You can also quickly select an option by typing its corresponding number.
*   **Intelligent Provider Selection**: Automatically detects and prioritizes available LLM providers based on configured API keys, falling back to others if a preferred one isn't configured.
*   **Automatic Staging**: If no files are staged, `kai` can automatically stage all changes in tracked files before generating a message.
*   **Contextual Commit History**: Can include previous commit messages for similar files in the prompt, helping the AI generate more consistent and contextually relevant messages.
*   **Multiple LLM Providers**: Supports various Large Language Model providers for flexibility:
    *   [Phind](https://www.phind.com/) (Default fallback)
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

Ensure that your `GOBIN` is in your system's `PATH` (e.g., `export PATH=$PATH:$(go env GOBIN)`) to run `kai` directly from your terminal.

### Building from source

Alternatively, you can clone the repository and build `kai` yourself. This method requires `make`.

```bash
git clone https://github.com/zbiljic/kai.git
cd kai
make install
```
The `make install` command will compile the `kai` executable and place it in your `GOBIN` directory.

## 💡 Usage

`kai` is designed to be used within a Git repository.

1.  **Stage your changes (or let `kai` do it):**
    Before running `kai`, you usually stage your changes:
    ```bash
    git add .
    ```
    However, if you forget to stage, `kai` will automatically stage all changes in tracked files by default (similar to `git add .`) before attempting to generate a message.

    To explicitly stage all changes and then generate a message, use the `--all` or `-a` flag:
    ```bash
    kai gen --all
    # or
    kai gen -a
    ```

2.  **Generate a commit message**:
    Simply run `kai` or `kai gen` in your repository:
    ```bash
    kai
    # or
    kai gen
    ```
    `kai` will analyze your staged changes, automatically detect an available LLM provider, generate potential commit messages, and present them in an interactive prompt:

    ```
    ◆ Pick a commit message to use: (Ctrl+c to exit)
      ● [1] feat: add new feature (e to edit)
      ○ [2] fix: resolve bug
      ○ [3] chore: update dependencies
    ```

    You can navigate through the suggestions using arrow keys. Press `Enter` to select a message.
    It is also possible to quickly pick a message by typing the corresponding number (e.g., `1`, `2`, `3`) instead of using arrow keys.

    Press `e` to edit the currently selected message interactively. If you choose to edit, `kai` will guide you through modifying the type, scope, and message body, especially useful for adhering to Conventional Commits.

3.  **Commit**: Once you select or confirm a message, `kai` will automatically commit your staged changes with the chosen message.

### Options

*   **Specify LLM Provider**: Use the `--provider` or `-p` flag to explicitly choose your desired LLM provider, overriding the automatic detection.
    ```bash
    kai gen --provider openai
    kai gen -p googleai
    ```
    Available providers: `phind` (default fallback), `openai`, `googleai`, `openrouter`.

*   **Specify Model**: Use the `--model` or `-m` flag to explicitly choose a specific model for the selected provider.
    ```bash
    kai gen --provider openai --model gpt-4
    kai gen -p googleai -m gemini-2.5-pro
    ```

*   **Specify Commit Message Type**: Use the `--type` or `-t` flag to set the desired commit message format.
    ```bash
    kai gen --type simple
    kai gen -t conventional
    ```
    Available types: `conventional` (default), `simple`.
    *   `conventional`: Generates messages adhering to the Conventional Commits specification (e.g., `type(scope): message`).
    *   `simple`: Generates plain messages like `message`.

*   **Include Previous Commit History**: By default, `kai` includes previous commit messages for relevant files to provide context to the AI. To disable this, use the `--history=false` flag:
    ```bash
    kai gen --history=false
    ```

*   **Number of Suggestions**: Use the `--count` or `-n` flag to specify how many commit message suggestions to generate (default is 2).
    ```bash
    kai gen --count 5
    ```

*   **Non-interactive Mode**: Use the `--yes` or `-y` flag to automatically use the first generated commit message without an interactive prompt.
    ```bash
    kai gen --yes
    ```

### `absorb` Command

The `absorb` command helps you automatically create `fixup!` commits for staged changes, targeting the original commits that introduced those changes. This is useful for splitting out and organizing your work.

```bash
kai absorb [options]
```

After running `kai absorb`, you can execute `git rebase -i --autosquash` to automatically squash the `fixup!` commits into their respective targets.

*   **Automatically Rebase**: Use `--and-rebase` or `-r` to automatically run `git rebase --autosquash` after creating fixups.
    ```bash
    kai absorb --and-rebase
    ```
*   **Dry Run**: Use `--dry-run` or `-n` to see what changes `absorb` would make without actually performing them.
    ```bash
    kai absorb --dry-run
    ```
*   **Backup Branch**: When using `--and-rebase`, use `--backup` or `-b` to create a backup branch before the rebase operation. This helps in recovery if the rebase fails or does not produce expected results.
    ```bash
    kai absorb --and-rebase --backup
    ```
*   **Stage All Changes**: Use `--all` or `-a` to automatically stage all changes in tracked files before analyzing for fixups.
    ```bash
    kai absorb --all
    ```

## ⚙️ Configuration

`kai` relies on environment variables for API keys to access LLM providers.

**Automatic Provider Selection:** `kai` will automatically detect and prioritize LLM providers based on the presence of their respective API keys in your environment variables. The preferred order of detection is:
1.  **Google AI**: Requires `GEMINI_API_KEY`
2.  **OpenRouter**: Requires `OPENROUTER_API_KEY`
3.  **OpenAI**: Requires `OPENAI_API_KEY`
4.  **Phind**: Does not require an API key (used as a last resort if others aren't configured).

To configure a provider, set the corresponding environment variable:

*   **OpenAI**:
    ```bash
    export OPENAI_API_KEY="your_openai_api_key"
    ```

*   **Google AI**:
    ```bash
    export GEMINI_API_KEY="your_google_ai_api_key"
    ```

*   **OpenRouter**:
    ```bash
    export OPENROUTER_API_KEY="your_openrouter_api_key"
    ```
    With OpenRouter, it is also possible to set these for custom attribution in their API logs:
    ```bash
    export OPENROUTER_HTTP_REFERER="https://github.com/zbiljic/kai"
    export OPENROUTER_X_TITLE="kai"
    ```

## 🤝 Contributing

Contributions are welcome! If you find a bug, have a feature request, or want to improve the codebase, please feel free to open an issue or submit a pull request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

*   Inspired by other AI-powered commit tools and the need for a simple Go solution that allows easy message modification and robust LLM integration.
*   Uses [go-clack](https://github.com/orochaa/go-clack) for interactive prompts.

## Similar Projects

Here are some other similar projects that you might find useful:

*   [aicommits](https://github.com/Nutlope/aicommits): A CLI that writes your Git commit messages for you with AI. (TypeScript)
*   [lumen](https://github.com/jnsahaj/lumen): Instant AI Git Commit message, Git changes summary from the CLI. (Rust)
