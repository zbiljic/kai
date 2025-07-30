# kai

[![Go Report Card](https://goreportcard.com/badge/github.com/zbiljic/kai)](https://goreportcard.com/report/github.com/zbiljic/kai)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`kai` is a command-line interface (CLI) tool that leverages Artificial Intelligence to automatically generate Git commit messages, pull request content, and even reorganize messy commit histories. It aims to streamline your development workflow by providing concise, relevant, and optionally Conventional Commit-formatted messages, allowing you to focus more on coding and less on crafting perfect commit messages or untangling commit history.

## ✨ Features

*   **AI-Powered Commit Message Generation (`gen`)**: Automatically creates commit messages by analyzing the diff of your staged Git changes.
*   **AI-Powered Pull Request Content Generation (`prgen`)**: Generates titles and descriptions for pull requests by analyzing commits and diffs between branches.
*   **AI-Powered Commit History Reorganization (`prprepare`)**: Analyzes your branch's full diff and uses AI to suggest and apply a clean, logical sequence of atomic commits, making code reviews easier.
*   **Intelligent Fixup (`absorb`)**: Automatically creates `fixup!` commits for staged changes, targeting the most appropriate original commits.
*   **Conventional Commits Support**: Generates commit messages adhering to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification (e.g., `feat(scope): add new feature`).
*   **Interactive Workflow**: Provides a selection of generated messages and allows interactive editing before committing. You can also quickly select an option by typing its corresponding number.
*   **Intelligent Provider Selection**: Automatically detects and prioritizes available LLM providers based on configured API keys, falling back to others if a preferred one isn't configured.
*   **Automatic Staging**: If no files are staged, `kai` can automatically stage all changes in tracked files before generating a message.
*   **Contextual Commit History**: Can include previous commit messages for similar files (including those in parent directories) in the prompt, helping the AI generate more consistent and contextually relevant messages.
*   **Multiple LLM Providers**: Supports various Large Language Model providers for flexibility:
    *   [Phind](https://www.phind.com/) (Default fallback)
    *   [OpenAI](https://openai.com/) (GPT-4o Mini, GPT-3.5 Turbo, etc.)
    *   [Anthropic](https://www.anthropic.com/) (Claude 3 Haiku, etc.)
    *   [Google AI](https://ai.google.dev/) (Gemini models)
    *   [OpenRouter](https://openrouter.ai/) (various models, including MistralAI)
    *   [Groq](https://groq.com/) (Llama models, Mixtral)
    *   [DeepSeek](https://deepseek.com/) (DeepSeek Chat, DeepSeek Coder, etc.)
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

### Generate Commit Message (`gen`)

This command generates Git commit messages based on your staged changes.

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

#### `gen` Options

*   **Specify LLM Provider**: Use the `--provider` or `-p` flag to explicitly choose your desired LLM provider, overriding the automatic detection.
    ```bash
    kai gen --provider openai
    kai gen -p googleai
    ```
    Available providers: `phind` (default fallback), `openai`, `claude`, `googleai`, `openrouter`, `groq`, `deepseek`.

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

*   **Include Previous Commit History**: By default, `kai` includes previous commit messages for relevant files (and their parent directories if no direct file history exists) to provide context to the AI. To disable this, use the `--history=false` flag:
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

### Generate Pull Request Content (`prgen`)

The `prgen` command helps you automatically generate a title and description for your pull request (PR) or merge request (MR) by analyzing the commits and changes between your current branch and a specified base branch.

```bash
kai prgen [options]
# or
kai pr [options]
```

To use it:

1.  **Switch to your feature branch**: Ensure you are on the branch for which you want to create a PR.
    ```bash
    git checkout feature/my-new-feature
    ```
2.  **Run `kai prgen`**:
    ```bash
    kai prgen
    ```
    `kai` will then:
    *   Compare your current branch with the default base branch (`main`).
    *   Optionally ask for additional context about your changes.
    *   Attempt to find and use a PR template in your repository.
    *   Generate a PR title and description based on the commits and diff.
    *   Display the generated content directly in your terminal.

#### `prgen` Options

*   **Specify LLM Provider**: Use the `--provider` or `-p` flag to explicitly choose your desired LLM provider, overriding the automatic detection.
    ```bash
    kai prgen --provider openai
    kai prgen -p googleai
    ```
    Available providers: `phind` (default fallback), `openai`, `claude`, `googleai`, `openrouter`, `groq`, `deepseek`.

*   **Specify Model**: Use the `--model` or `-m` flag to explicitly choose a specific model for the selected provider.
    ```bash
    kai prgen --provider openai --model gpt-4-turbo
    kai prgen -p googleai -m gemini-pro
    ```

*   **Specify Base Branch**: Use the `--base` or `-b` flag to compare against a branch other than `main`.
    ```bash
    kai prgen --base develop
    ```

*   **Maximum Diff Size**: Use `--max-diff` to set a limit (in characters) on the size of the code diff sent to the LLM. Larger diffs consume more tokens and might be truncated by some models.
    ```bash
    kai prgen --max-diff 5000
    ```
    The default is 10000 characters.

*   **No Additional Context Prompt**: Use `--no-context` to skip the interactive prompt for additional business or feature context. The AI will rely solely on the commit messages and code diff.
    ```bash
    kai prgen --no-context
    ```

### Reorganize Commit History (`prprepare`)

The `prprepare` command uses AI to analyze your current branch's entire diff against a base branch and suggest a reorganized, cleaner commit history. It helps you transform messy, large, or poorly structured commits into logical, atomic units, which significantly improves code review readability and maintainability.

```bash
kai prprepare [options]
# or
kai prp [options]
```

To use it:

1.  **Switch to your feature branch**: Ensure you are on the branch whose history you want to reorganize.
    ```bash
    git checkout feature/my-complex-feature
    ```
2.  **Run `kai prprepare`**:
    ```bash
    kai prprepare
    ```
    `kai` will then:
    *   Analyze all changes between your current branch and the default base branch (`main`).
    *   Parse the diff into individual "hunks" of changes.
    *   Use an LLM to generate a proposed commit plan, detailing new commit messages and which specific hunks belong to each new commit.
    *   Display the proposed plan for your review.
    *   If confirmed, it will **reset your branch to the base branch** and then apply the new commits sequentially, rebuilding your history.

    **Important**: This command rewrites your branch's history. It's recommended to have a backup or ensure your work is pushed before running it, especially without `--dry-run`. `kai` will create a temporary backup branch by default before applying changes if `--debug` or `--auto-apply` is not used.

#### `prprepare` Options

*   **Specify LLM Provider**: Use the `--provider` or `-p` flag to explicitly choose your desired LLM provider, overriding the automatic detection.
    ```bash
    kai prprepare --provider claude
    kai prprepare -p deepseek
    ```
    Available providers: `phind` (default fallback), `openai`, `claude`, `googleai`, `openrouter`, `groq`, `deepseek`.

*   **Specify Model**: Use the `--model` or `-m` flag to explicitly choose a specific model for the selected provider.
    ```bash
    kai prprepare --provider claude --model claude-3-opus-20240229
    kai prprepare -p groq -m mixtral-8x7b-32768
    ```

*   **Specify Base Branch**: Use the `--base` or `-b` flag to compare against a branch other than `main`.
    ```bash
    kai prprepare --base feature/my-base
    ```

*   **Maximum Diff Size**: Use `--max-diff` to set a limit (in characters) on the total size of the code diff sent to the LLM for analysis. This helps manage token usage for very large diffs.
    ```bash
    kai prprepare --max-diff 20000
    ```
    The default is 10000 characters.

*   **Automatically Apply**: Use `--auto-apply` to skip the confirmation prompt and immediately apply the generated commit reorganization plan. **Use with caution!**
    ```bash
    kai prprepare --auto-apply
    ```

*   **Dry Run**: Use `--dry-run` or `-n` to simulate the reorganization without making any actual changes to your repository. It will show the proposed plan and what `git` commands would be executed.
    ```bash
    kai prprepare --dry-run
    ```

*   **Debug Mode**: Use `--debug` to enable detailed logging and write each proposed commit's patch file to `.kai/prprepare/` in your repository. This implies `--dry-run` if used alone. Useful for inspecting the AI's hunk grouping and patch creation.
    ```bash
    kai prprepare --debug
    ```
    When debug is enabled, you can then manually apply the patches (e.g., `git apply --check --cached .kai/prprepare/001.patch`).

### Absorb Staged Changes (`absorb`)

The `absorb` command helps you automatically create `fixup!` commits for staged changes, targeting the original commits that introduced those changes. This is useful for splitting out and organizing your work and for making small corrections to previous commits before a final rebase.

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
*   **Backup Branch**: When using `--and-rebase`, use `--backup` or `-b` to create a backup branch (e.g., `backup/your-branch-HH-MM-SS`) before the rebase operation. This helps in recovery if the rebase fails or does not produce expected results. `kai` will check if an up-to-date backup exists and reuse it if possible.
    ```bash
    kai absorb --and-rebase --backup
    ```
*   **Stage All Changes**: Use `--all` or `-a` to automatically stage all changes in tracked files before analyzing for fixups.
    ```bash
    kai absorb --all
    ```
*   **Maximum History Lookback**: Use `--max-history` to specify how many commits back `absorb` should look when trying to find the original commit for a modified line (default is 20).
    ```bash
    kai absorb --max-history 50
    ```

## ⚙️ Configuration

`kai` relies on environment variables for API keys to access LLM providers.

**Automatic Provider Selection:** `kai` will automatically detect and prioritize LLM providers based on the presence of their respective API keys in your environment variables. The preferred order of detection (most preferred first) is:
1.  **Google AI**: Requires `GEMINI_API_KEY`
2.  **Groq**: Requires `GROQ_API_KEY`
3.  **OpenRouter**: Requires `OPENROUTER_API_KEY`
4.  **OpenAI**: Requires `OPENAI_API_KEY`
5.  **Anthropic Claude**: Requires `ANTHROPIC_API_KEY`
6.  **DeepSeek**: Requires `DEEPSEEK_API_KEY`
7.  **Phind**: Does not require an API key (used as a last resort if others aren't configured).

To configure a provider, set the corresponding environment variable:

*   **Google AI**:
    ```bash
    export GEMINI_API_KEY="your_google_ai_api_key"
    ```

*   **Groq**:
    ```bash
    export GROQ_API_KEY="your_groq_api_key"
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

*   **OpenAI**:
    ```bash
    export OPENAI_API_KEY="your_openai_api_key"
    ```

*   **Anthropic Claude**:
    ```bash
    export ANTHROPIC_API_KEY="your_anthropic_api_key"
    ```

*   **DeepSeek**:
    ```bash
    export DEEPSEEK_API_KEY="your_deepseek_api_key"
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
*   [prghost](https://github.com/fyvfyv/prghost): Tool that auto-generates PR descriptions from git diffs using AI and project guidelines. (TypeScript)
