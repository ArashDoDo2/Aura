package main

/*
#cgo CFLAGS: -std=c99
#include <stdlib.h>
*/
import "C"

import (
	"log"
	"unsafe"

	"github.com/ArashDoDo2/Aura/internal"
)

//export StartAura
func StartAura(dnsServer *C.char, domain *C.char) {
	goDNS := C.GoString((*C.char)(unsafe.Pointer(dnsServer)))
	goDomain := C.GoString((*C.char)(unsafe.Pointer(domain)))

	if goDomain == "" {
		log.Printf("StartAura: domain is empty")
		return
	}

	if err := internal.StartAuraClient(goDNS, goDomain, 1080); err != nil {
		log.Printf("StartAura error: %v", err)
	}
}

//export StopAura
func StopAura() {
	internal.StopAuraClient()
}

func main() {}
