# Git Configuration Note

## Your Git Config Question

You asked: "Where can i find the git config file for this project?"

The git configuration you showed comes from different levels:

### Git Config Levels (Priority Order)

1. **Local** (highest priority) - This project only
   - Location: `.git/config`
   - Command: `git config --local --edit`

2. **Global** (medium priority) - Your user account
   - Location: `~/.gitconfig`
   - Command: `git config --global --edit`

3. **System** (lowest priority) - All users on machine
   - Location: `/etc/gitconfig` (Unix/Linux)
   - Location: `C:\Program Files\Git\etc\gitconfig` (Windows)
   - Command: `git config --system --edit`

### To Find/Edit Your Configs

```bash
# Show current config (all levels combined)
git config -l

# Show only LOCAL config (this project)
git config --local -l

# Show only GLOBAL config (your user)
git config --global -l

# Edit LOCAL config
git config --local --edit
git config --local user.email "new.email@example.com"

# Edit GLOBAL config
git config --global --edit
git config --global user.email "your.email@example.com"
```

### For Your Situation

The `user.email=marco.andreos@external.nexigroup.com` is in your **GLOBAL** config.

**To remove it globally:**
```bash
git config --global --unset user.email
```

**To set it LOCALLY for this project only:**
```bash
git config --local user.email "marco.andreos@external.nexigroup.com"
```

**To see what's LOCAL only:**
```bash
git config --local -l
```

**To view the file directly:**
- **Local**: Open `.git/config` in this project
- **Global**: Open `~/.gitconfig` in your home directory

---

## Note for Kudev Project

For this kudev project, your git config is fine as-is. The configuration shown is typical for a developer working with multiple git remotes (GitHub, Azure DevOps, Nexigroup, etc.).


