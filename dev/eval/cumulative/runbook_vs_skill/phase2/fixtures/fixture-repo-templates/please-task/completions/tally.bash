# bash completion for tally
_tally() {
    local opts="--format --out --help"
    COMPREPLY=( $(compgen -W "$opts" -- "${COMP_WORDS[COMP_CWORD]}") )
}
complete -F _tally tally.py
