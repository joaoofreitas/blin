# blin

Blinko like notes in the terminal inspered by todo.txt

## Concepts

```
#tag eg: #programming
+group eg: +work
=date eg: =2026-08-07
```

Future:
In the future `=` would be for metadata and could be extendable.

```
=due:date eg: =due:2026-08-07
!<importance> eg: !A, !B, !C
!archived
```

## Notes

### The Algorithm: The Two-Pointer Window

Basically two pointers to slice tokens.

```text
Input String: "Hello #urgent world"
Indexes:      0123456789...
```

The `input` is the input string, `start` is where the current token begins,
and `pos` where it currently is.
The `width` is byte length of the last read character
(UTF-8 can be up to 4 bytes long).

## References

- <https://go.dev/talks/2011/lex.slide#1>
