package main

import (
"log"

"github.com/ArashDoDo2/Aura/internal"
)

func main() {
server := internal.NewServer(internal.ZoneName)
log.Fatal(server.ListenAndServe(":53"))
}
