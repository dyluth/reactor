# Shell Completions with `reactor completion`

Generate shell completion scripts for enhanced command-line experience.

## Installation

**Bash**:
```bash
# Add to current session
source <(reactor completion bash)

# Install permanently
reactor completion bash > ~/.bash_completion.d/reactor
# or
echo 'source <(reactor completion bash)' >> ~/.bashrc
```

**Zsh**:
```bash
# Add to current session
source <(reactor completion zsh)

# Install permanently (ensure ~/.zsh/completions is in $fpath)
reactor completion zsh > ~/.zsh/completions/_reactor
# or
echo 'source <(reactor completion zsh)' >> ~/.zshrc
```

**Fish**:
```bash
# Install permanently
reactor completion fish | source
# or save to file
reactor completion fish > ~/.config/fish/completions/reactor.fish
```

## Features

With completions installed, you get:

- **Command completion**: `reactor <TAB>` shows available commands
- **Flag completion**: `reactor run --<TAB>` shows available flags  
- **Value completion**: Context-aware suggestions for flag values
- **Help integration**: Descriptions appear during completion

## Verification

Test completions work:
```bash
reactor <TAB><TAB>        # Should show: run, diff, sessions, config, accounts, completion
reactor run --<TAB><TAB>  # Should show: --image, --account, --discovery-mode, etc.
```