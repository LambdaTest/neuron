#!/usr/bin/env bash

# handy color vars for pretty prompts
BLACK="\033[0;30m"
BLUE="\033[0;34m"
GREEN="\033[0;32m"
CYAN="\033[0;36m"
RED="\033[0;31m"
PURPLE="\033[0;35m"
BROWN="\033[0;33m"
WHITE="\033[1;37m"
COLOR_RESET="\033[0m"


function check_dependecy() {
    command -v $1 >/dev/null 2>&1 || {
        
        echo ""
        echo -e "${RED}##############################################################"
        echo "# HOLD IT!! I require $1 but it's not installed.  Aborting." >&2;
        echo -e "${RED}##############################################################"
        echo ""
        echo -e "${COLOR_RESET}Installing $1:"
        echo ""
        echo -e "${BLUE}Mac:${COLOR_RESET} $ brew install $1"
        echo ""
        echo -e "${BLUE}Other:${COLOR_RESET} $2"
        echo -e "${COLOR_RESET}"
        exit 1;
    }
}

declare -a arr=("kubectl" "https://kubernetes.io/docs/tasks/tools/" "helm" "https://helm.sh/docs/intro/install/" "vault" "https://learn.hashicorp.com/tutorials/vault/getting-started-install" "jq" "https://github.com/stedolan/jq/wiki/Installation")
l=${#arr[@]}

for (( i=0; i<${l}; i+=2 ));
do
    check_dependecy "${arr[$i]}" "${arr[$i+1]}"
done

