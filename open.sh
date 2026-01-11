#!/bin/bash

# if this isn't macos crash out
if [[ "$(uname)" != "Darwin" ]]; then
  echo "This script is only for macOS"
  exit 1
fi

# if firefox isn't installed crash out
if [ ! -d "/Applications/Firefox.app" ]; then
  echo "Firefox is not installed in /Applications"
  exit 1
fi

go build /Users/sam/git/samiam2013/pugnarehealth

./pugnarehealth --skip-update-check || { echo "Build or execution failed"; exit 1; }


echo "opening local index.html in Firefox"

/Applications/Firefox.app/Contents/MacOS/firefox --new-tab "file:///Users/sam/git/samiam2013/pugnarehealth/public/index.html"
