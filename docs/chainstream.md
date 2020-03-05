# Chain Stream

Chain stream is a way of passing changes to a blockchain to a listener without running fork choice. More specifically, this shows how to translate a stream of block actions to a series of ADD_BLOCK and ROLLBACK_BLOCK.

```
A -> B -> C
     |
      \-> D -> E
```

Say we have the following chain where A is justified and E is the current chain tip. This means that the following chain stream calls have been executed prior:

```python
action(A, true, nil)
action(B, false, A)
action(C, false, B)
action(D, false, B)
action(E, false, D)
```

This can be translated into the following actions for a state database:
```python
execute(A)
commit(A)
execute(B)
execute(C)
rollback(C)
execute(D)
execute(E)
```