#!/usr/bin/env bash

# Checkout
git checkout --orphan latest_branch

# Add all the files
git add -A

# Commit the changes
git commit -am "reinit commit"

# Delete the branch
git branch -D main

# Rename the current branch to main
git branch -m main

# Finally, force update your repository
git push -f origin main
