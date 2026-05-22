## rshred v3.0.0 May 18 2026

- warned about invalid directory paths placed in config file

- remove IFS for inital path input to protect leading and trailing whitespace

- added config file versioning

- fixed directory exclusion

- fixed "cutsom" typo

- fixed error detection after shredding

- renamed variables for clarity

- restructured log path validation

- unified `realpath` calls

- edited warning about faliure to remove tempfiles

- updated keys in config file for clarity

- fixed bug that prevented certain symbols from being able to be in an excluded directory 

## rshred v2.1.0 March 11 2026

-added update detection

-added option to remove tempfiles instead of shredding them

-fixed default temp folder when no config file is set

## rshred v2.0.0 November 25 2025
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
