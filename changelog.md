## rshred v2.0.0 (Unreleased)
- added directory exclusion

- added custom tempfile selection

- added warning about invalid logging directory

- added SIGHUP to valid signals to be trapped

- added option to recursively delete folders within shredding directory

- prioritized tempfiles instead of logfiles for error scanning

- unified config file locations under two variables

- added color to some errors and warnings

- fixed bug preventing logging

- used `realpath` to output paths in final messages instead of calling the shred target variable directly

- clarified valid options and defaults in config file

- redesigned config loading mechanism from `source` to a `while` loop


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
