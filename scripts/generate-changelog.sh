#!/bin/bash

# Script to generate changelog between two git tags
# Usage: ./scripts/generate-changelog.sh [from-tag] [to-tag]
# Example: ./scripts/generate-changelog.sh v0.1.0 v0.2.0
# If no tags provided, it will use the latest tag and HEAD

set -e

# Function to get the latest tag
get_latest_tag() {
    git describe --tags --abbrev=0 2>/dev/null || echo ""
}

# Function to get the previous tag
get_previous_tag() {
    local current_tag=$1
    if [ -z "$current_tag" ]; then
        # If no current tag, get the latest tag
        git describe --tags --abbrev=0 2>/dev/null || echo ""
    else
        # Get the tag before the current one
        git describe --tags --abbrev=0 "$current_tag^" 2>/dev/null || echo ""
    fi
}

# Function to format commit message
format_commit() {
    local commit_hash=$1
    local commit_message=$2
    local commit_author=$3
    local commit_date=$4
    
    # Extract type and scope from conventional commit format
    if [[ $commit_message =~ ^([a-z]+)(\([^)]+\))?: ]]; then
        local type="${BASH_REMATCH[1]}"
        local scope="${BASH_REMATCH[2]}"
        local description="${commit_message#*: }"
        
        case $type in
            feat|feature)
                echo "- ✨ **Feature**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            fix)
                echo "- 🐛 **Fix**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            docs)
                echo "- 📚 **Documentation**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            style)
                echo "- 💄 **Style**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            refactor)
                echo "- ♻️ **Refactor**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            perf)
                echo "- ⚡ **Performance**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            test)
                echo "- 🧪 **Test**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            build)
                echo "- 🔨 **Build**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            ci)
                echo "- 👷 **CI**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            chore)
                echo "- 🔧 **Chore**: $description ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
            *)
                echo "- 📝 **Other**: $commit_message ([${commit_hash:0:7}](../../commit/$commit_hash))"
                ;;
        esac
    else
        # Not conventional commit format
        echo "- 📝 $commit_message ([${commit_hash:0:7}](../../commit/$commit_hash))"
    fi
}

# Function to categorize commits
categorize_commits() {
    local from_tag=$1
    local to_tag=$2
    
    # Arrays to store different types of commits
    declare -a features=()
    declare -a fixes=()
    declare -a breaking=()
    declare -a docs=()
    declare -a others=()
    
    # Get commits between tags
    local range
    if [ -z "$from_tag" ]; then
        range="$to_tag"
    else
        range="$from_tag..$to_tag"
    fi
    
    # Read commits
    while IFS='|' read -r hash message author date; do
        if [[ $message =~ ^([a-z]+)(\([^)]+\))?: ]]; then
            local type="${BASH_REMATCH[1]}"
            local description="${message#*: }"
            
            # Check for breaking changes
            if [[ $message == *"BREAKING CHANGE"* ]] || [[ $message == *"!"* ]]; then
                breaking+=("$(format_commit "$hash" "$message" "$author" "$date")")
            elif [[ $type == "feat" || $type == "feature" ]]; then
                features+=("$(format_commit "$hash" "$message" "$author" "$date")")
            elif [[ $type == "fix" ]]; then
                fixes+=("$(format_commit "$hash" "$message" "$author" "$date")")
            elif [[ $type == "docs" ]]; then
                docs+=("$(format_commit "$hash" "$message" "$author" "$date")")
            else
                others+=("$(format_commit "$hash" "$message" "$author" "$date")")
            fi
        else
            others+=("$(format_commit "$hash" "$message" "$author" "$date")")
        fi
    done < <(git log --format="%H|%s|%an|%ad" --date=short "$range" --reverse)
    
    # Generate changelog
    echo "# Changelog"
    echo ""
    
    if [ -n "$to_tag" ]; then
        echo "## [$to_tag] - $(date +%Y-%m-%d)"
    else
        echo "## [Unreleased] - $(date +%Y-%m-%d)"
    fi
    echo ""
    
    # Breaking changes first
    if [ ${#breaking[@]} -gt 0 ]; then
        echo "### 💥 Breaking Changes"
        echo ""
        for commit in "${breaking[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    # Features
    if [ ${#features[@]} -gt 0 ]; then
        echo "### ✨ Features"
        echo ""
        for commit in "${features[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    # Bug fixes
    if [ ${#fixes[@]} -gt 0 ]; then
        echo "### 🐛 Bug Fixes"
        echo ""
        for commit in "${fixes[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    # Documentation
    if [ ${#docs[@]} -gt 0 ]; then
        echo "### 📚 Documentation"
        echo ""
        for commit in "${docs[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    # Other changes
    if [ ${#others[@]} -gt 0 ]; then
        echo "### 🔧 Other Changes"
        echo ""
        for commit in "${others[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    # Add contributors
    echo "### 👥 Contributors"
    echo ""
    git log --format="%an" "$range" | sort | uniq | while read -r author; do
        echo "- $author"
    done
    echo ""
    
    # Add comparison link if both tags exist
    if [ -n "$from_tag" ] && [ -n "$to_tag" ]; then
        local repo_url=$(git config --get remote.origin.url | sed 's/\.git$//' | sed 's/git@github.com:/https:\/\/github.com\//')
        echo "**Full Changelog**: [$from_tag...$to_tag]($repo_url/compare/$from_tag...$to_tag)"
    fi
}

# Main script logic
FROM_TAG=""
TO_TAG=""

if [ $# -eq 0 ]; then
    # No arguments, use latest tag to HEAD
    TO_TAG="HEAD"
    FROM_TAG=$(get_latest_tag)
    if [ -z "$FROM_TAG" ]; then
        echo "No tags found in repository. Please create a tag first or specify tags manually."
        exit 1
    fi
    echo "Generating changelog from $FROM_TAG to HEAD..."
elif [ $# -eq 1 ]; then
    # One argument, use it as TO_TAG and find previous tag
    TO_TAG=$1
    FROM_TAG=$(get_previous_tag "$TO_TAG")
    if [ -z "$FROM_TAG" ]; then
        echo "No previous tag found before $TO_TAG. Generating changelog from beginning..."
    else
        echo "Generating changelog from $FROM_TAG to $TO_TAG..."
    fi
elif [ $# -eq 2 ]; then
    # Two arguments, use them as FROM and TO tags
    FROM_TAG=$1
    TO_TAG=$2
    echo "Generating changelog from $FROM_TAG to $TO_TAG..."
else
    echo "Usage: $0 [from-tag] [to-tag]"
    echo "Examples:"
    echo "  $0                    # Latest tag to HEAD"
    echo "  $0 v0.2.0            # Previous tag to v0.2.0"
    echo "  $0 v0.1.0 v0.2.0     # v0.1.0 to v0.2.0"
    exit 1
fi

# Verify tags exist
if [ -n "$FROM_TAG" ] && [ "$FROM_TAG" != "HEAD" ] && ! git rev-parse "$FROM_TAG" >/dev/null 2>&1; then
    echo "Error: Tag '$FROM_TAG' does not exist"
    exit 1
fi

if [ -n "$TO_TAG" ] && [ "$TO_TAG" != "HEAD" ] && ! git rev-parse "$TO_TAG" >/dev/null 2>&1; then
    echo "Error: Tag '$TO_TAG' does not exist"
    exit 1
fi

# Generate the changelog
categorize_commits "$FROM_TAG" "$TO_TAG"