## rshred v1.1.0 (Unreleased)

- included warning about invalid logging directory

- fixed catastrophic bug preventing logging

- used `realpath` to output paths in final messages instead of calling the shred target variable directly


## rshred v1.0.4 November 15 2025
- edited final confirmation message for clarity

- added final confirmation message specific to single-file shredding

- added "Shred aborted." message when final confirmation fails

- removed manual tempfile shredding code after shred is aborted after permission check and called shredtemps function instead 

- disabled execution permission on script by default

## rshred v1.0.3 November 12 2025

- removed calls to unset vestigial variables

- removed "c" from accepted characters in flag prompt

## rshred v1.0.2 August 9 2025

- changed all tilde instances in echo statements to $HOME for user clarity

- changed shebang to "/usr/bin/env bash"

## rshred v1.0.1 August 8 2025

- fixed bug involving how the -s flag is handled
