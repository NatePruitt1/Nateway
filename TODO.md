# Todo list for gateway.

### Actual Bugfixes
    - [x] Bodies are never closed in the request cycle.
    - [ ] Configuration rework and validation
        - Move from a file edit trigger (specifically locking the file once in use) to a cli
        swap command, which can tell the user what will happen, and allowing for A/B swapping 
        including the reloads, so that the old config can drain for the new one. Everything 
        will be contained in subthreads from the CLI, so that reloads do not require a process
        kill. 
        - [ ] Build a simple CLI language parser for reloads.
        - [ ] Build a server goroutine inside of main that contains all the resources that a 
        server needs (and defers the correct cleanup routine). This requires a move to use 
        no log.Fatal calls and reducing global variable usage, as they will race in the drain 
        phase. 
        - [ ] Build clean swapping logic into the CLI/Manager which allows the user to know
        how things will work and if a restart and swap will happen.
    - [ ] Check redis is reachable in startup, and throw error if it does not connect.
    - [ ] Hotreloads leek a redis connection
        - Close and reopen should be the easiest.

### Making a solid load test.
    - [ ] Set up a proper echo server.
    - [ ] Check status codes in the test and make sure everything is wired correctly.
    - [ ] Write up the k6 to do proper testing and data storage.

### Low severtiy fixes.
    - [ ] Fix vet warnings (tokenstring=tokenstring in benchtest.go)
    - [ ] Move context endpoints to a typed string.

### Testing needs.
    - [ ] Test that response body leak stays fixed.
    - [ ] Test that config changes work.