#!/bin/bash

set -o nounset

CHANGELOG="./CHANGELOG.md"
EDITOR=${EDITOR:-vi}

# Matches vMAJOR.MINOR.PATCH-gitlab
RELEASE_PATTERN='^v[0-9]+\.[0-9]+\.[0-9]+\-gitlab$'
All_RELEASE_TAGS=$(git tag | grep -E "$RELEASE_PATTERN" | sort --version-sort)
LATEST_RELEASE_TAG=$(echo "$All_RELEASE_TAGS" | tail -n1)

# Suggests incrementing by one patch relase.
SUGGESTED_PATCH_RELEASE_TAG=$(echo "$LATEST_RELEASE_TAG" | awk -F "[\.\-]" '{$3=$3+1; print $1"."$2"."$3"-gitlab"}')

LATEST_COMMIT_MADE_BY_SCRIPT=false

confirm() {
  message=$1

while true; do
  read -r -p "$message [Y/N]: " answer
  case $answer in
    [Yy]* )
      return 0;;
    [Nn]* )
      return 1;;
    * )
      echo "Please answer [Y]es or [N]o";;
  esac
done
}

while true; do
  # Ask user for release tag version.
  read -r -p "Enter the release version [$SUGGESTED_PATCH_RELEASE_TAG]: " tag
  RELEASE_TAG=${tag:-$SUGGESTED_PATCH_RELEASE_TAG}

  # Ensure the user's tag conforms to our release pattern.
  if ! grep -E "$RELEASE_PATTERN" <<< "$RELEASE_TAG"; then
    echo "Relase version must be in the form vMAJOR.MINOR.PATCH-gitlab";
    continue;
  fi

  # Ensure the tag has not been previously used.
  if grep "$RELEASE_TAG" <<< "$All_RELEASE_TAGS"; then
    echo "Release version must not be a previously released version:";
    echo "$All_RELEASE_TAGS";
    continue;
  fi

  # Tag conforms, continue.
  break;
done

# If the changelog doesn't have an entry for the release tag, prepend an
# example entry and open the changelog in EDITOR.
if ! grep "$RELEASE_TAG" "$CHANGELOG"; then
  sed -i "1s;^;##$RELEASE_TAG\n\n\- Add your changes here\n\n;" "$CHANGELOG"

  $EDITOR $CHANGELOG
fi

# If there are changes to the changelog, prompt the user to write a commit
# with a default message.
if ! git diff --quiet "$CHANGELOG"; then
  git add $CHANGELOG
  git commit --verbose --verbose --message="Prepare $RELEASE_TAG" --edit
  LATEST_COMMIT_MADE_BY_SCRIPT=true
fi

# Tag the release, allowing the user to edit a default message further.
# TODO annotated tags don't have a commit template, so getting dropped into
# this screen can be pretty jaring. It would be good to eventually include one.
git tag --annotate "$RELEASE_TAG" --message="Release $RELEASE_TAG" --edit

# Confirm the changes.
git show "$RELEASE_TAG"
echo
echo

if ! confirm "Please confirm the above changes, on confirmation they will be pushed upstream. If they are not confirmed, these changes will be reverted."; then
  git tag --delete "$RELEASE_TAG"

  # If it was commited by us, undo the most recent commit, keeping the changes on disk, but unstaged.
  if $LATEST_COMMIT_MADE_BY_SCRIPT; then
    git reset HEAD~
  fi
  exit 1
fi

# Push up the commit with the release tag and the tag itself.
git push "$RELEASE_TAG"
