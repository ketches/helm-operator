#!/bin/bash

# Script to generate release notes for GitHub releases
# Usage: ./scripts/generate-release-notes.sh [from-tag] [to-tag]

set -e

get_latest_tag() {
    git describe --tags --abbrev=0 2>/dev/null || echo ""
}

get_previous_tag() {
    local current_tag=$1
    if [ -z "$current_tag" ]; then
        git describe --tags --abbrev=0 2>/dev/null || echo ""
    else
        git describe --tags --abbrev=0 "$current_tag^" 2>/dev/null || echo ""
    fi
}

generate_release_notes() {
    local from_tag=$1
    local to_tag=$2
    
    declare -a features=()
    declare -a fixes=()
    declare -a others=()
    
    local range
    if [ -z "$from_tag" ]; then
        range="$to_tag"
    else
        range="$from_tag..$to_tag"
    fi
    
    while IFS='|' read -r hash message author date; do
        if [[ $message == "Merge "* ]]; then
            continue
        fi
        
        local formatted_line="- $message ([${hash:0:7}](../../commit/$hash))"
        
        if [[ $message == "feat:"* ]] || [[ $message == "feature:"* ]]; then
            features+=("$formatted_line")
        elif [[ $message == "fix:"* ]]; then
            fixes+=("$formatted_line")
        else
            others+=("$formatted_line")
        fi
    done < <(git log --format="%H|%s|%an|%ad" --date=short "$range" --reverse --no-merges)
    
    echo "## What's Changed"
    echo ""
    
    if [ ${#features[@]} -gt 0 ]; then
        echo "### New Features"
        for commit in "${features[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    if [ ${#fixes[@]} -gt 0 ]; then
        echo "### Bug Fixes"
        for commit in "${fixes[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    if [ ${#others[@]} -gt 0 ]; then
        echo "### Other Changes"
        for commit in "${others[@]}"; do
            echo "$commit"
        done
        echo ""
    fi
    
    echo "### Contributors"
    git log --format="%an" "$range" --no-merges | sort | uniq | while read -r author; do
        echo "- $author"
    done
    echo ""
    
    if [ -n "$from_tag" ] && [ -n "$to_tag" ]; then
        local repo_url=$(git config --get remote.origin.url | sed 's/\.git$//' | sed 's/git@github.com:/https:\/\/github.com\//')
        echo "**Full Changelog**: $repo_url/compare/$from_tag...$to_tag"
    fi
}

FROM_TAG=""
TO_TAG=""

if [ $# -eq 0 ]; then
    TO_TAG="HEAD"
    FROM_TAG=$(get_latest_tag)
elif [ $# -eq 1 ]; then
    TO_TAG=$1
    FROM_TAG=$(get_previous_tag "$TO_TAG")
elif [ $# -eq 2 ]; then
    FROM_TAG=$1
    TO_TAG=$2
else
    echo "Usage: $0 [from-tag] [to-tag]"
    exit 1
fi

generate_release_notes "$FROM_TAG" "$TO_TAG"
