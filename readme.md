# dua - disk usage analyzer

dua is a simple commandline utility, that scans the target directory
for files and directories taking up the most space.

It employs a basic heuristic (a threshold), to decide when to list a
single big file rather than its parent directory, so (unlike a
simplistic variation on `du | sort`) in many cases it correctly
identifies both large files, and directories with a lot of smaller
files that add up to a larger total.

```
dua [-h] [-t THRESHOLD] [-n N] <DIRECTORY>
```

Options:

- `-t THRESHOLD`: Set the threshold (default: 0.9; range (0.0 - 1.0)).
- `-n N`: Show top N results (default: 20).

## Author

&copy; 2023 Kamil Cholewiński <<kamil@rollc.at>>

License is [MIT](/license.txt).
