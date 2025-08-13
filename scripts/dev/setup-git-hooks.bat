@echo off
echo Setting up Git hooks...

:: Create hooks directory if it doesn't exist
if not exist ".git\hooks" (
    echo Creating .git\hooks directory...
    mkdir ".git\hooks"
)

:: Create pre-commit hook
echo Creating pre-commit hook...
(
echo #!/bin/sh
echo # Pre-commit hook to prevent committing sensitive files
echo.
echo # Check for AI assistant files
echo if git diff --cached --name-only ^| grep -E "^(\.claude^|\.anthropic^|\.cursor^|\.aider^|claude\.json^|\.copilot)"; then
echo     echo "ERROR: AI assistant files detected!"
echo     echo "These files should not be committed:"
echo     git diff --cached --name-only ^| grep -E "^(\.claude^|\.anthropic^|\.cursor^|\.aider^|claude\.json^|\.copilot)"
echo     echo.
echo     echo "Use 'git rm --cached ^<filename^>' to unstage them."
echo     exit 1
echo fi
echo.
echo # Check for environment files
echo if git diff --cached --name-only ^| grep -E "\.env$"; then
echo     echo "WARNING: Environment files detected!"
echo     echo "Make sure these don't contain sensitive information:"
echo     git diff --cached --name-only ^| grep -E "\.env$"
echo     echo.
echo     echo "Press Ctrl+C to cancel, or Enter to continue..."
echo     read
echo fi
echo.
echo exit 0
) > .git\hooks\pre-commit

echo.
echo Git hooks setup completed!
echo.
echo The pre-commit hook will now prevent accidental commits of:
echo - .claude files and directories
echo - .anthropic files
echo - .cursor files
echo - .aider files
echo - .copilot files
echo - claude.json
echo.
pause