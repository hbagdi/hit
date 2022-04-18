#!/usr/bin/env bash
_hit_completions()
{
  COMPREPLY=($(compgen -W "$(hit c1)" -- "${COMP_WORDS[1]}"))
}

complete -F _hit_completions hit
