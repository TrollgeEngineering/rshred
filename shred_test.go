package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestShred(t *testing.T) {
	test1Payload := []string{
		"Hello, this is a test file to test the shredding function.",
		"The purpose of this is to test the functions of the shred function such as shredding at different depths and progress reporting.",
	}
	var written uint64
	test2Payload := make([]byte, 8*(1<<20))
	for b := range test2Payload {
		test2Payload[b] = 0x50
	}
	test3Payload := make([]byte, 45*(1<<20))
	for b := range test3Payload {
		test3Payload[b] = 0xa0
	}
	if err := os.Mkdir("shred_tests", 0o770); err != nil {
		panic("failed to create testing directory shred_tests")
	}
	test1, openErr := os.OpenFile("shred_tests/test1.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o770)
	if openErr != nil {
		panic(fmt.Sprintf("failed to create test file at shred_tests/test1.txt: %s", openErr))
	}
	test2, openErr := os.OpenFile("shred_tests/test2.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o770)
	if openErr != nil {
		panic(fmt.Sprintf("failed to create test file at shred_tests/test2.txt: %s", openErr))
	}
	test3, openErr := os.OpenFile("shred_tests/test3.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o770)
	if openErr != nil {
		panic(fmt.Sprintf("failed to create test file at shred_tests/test3.txt: %s", openErr))
	}
	fmt.Println("filling shred_tests/test1.txt with test data...")
	for _, line := range test1Payload {
		if _, writeErr := fmt.Fprintln(test1, line); writeErr != nil {
			panic(fmt.Sprintf("failed to write to file at shred_tests/test1.txt: %s", writeErr))
		}
	}
	if _, seekErr := test1.Seek(0, 0); seekErr != nil {
		panic(fmt.Sprintf("failed to seek to 0 on file at shred_tests/test1.txt: %s", seekErr))
	}
	fmt.Println("filling shred_tests/test2.txt with test data...")
	for written < 9*(1<<30) {
		wrote, writeErr := test2.Write(test2Payload)
		if writeErr != nil {
			panic(fmt.Sprintf("failed to write to file at shred_tests/test2.txt: %s", writeErr))
		}
		written += uint64(wrote)
	}
	written = 0
	fmt.Println("syncing to test file at shred_tests/test2.txt...")
	if syncErr := test2.Sync(); syncErr != nil {
		panic(fmt.Sprintf("failed to sync file at shred_tests/test2.txt: %s", syncErr))
	}
	if _, seekErr := test2.Seek(0, 0); seekErr != nil {
		panic(fmt.Sprintf("failed to seek to 0 on file at shred_tests/test2.txt: %s", seekErr))
	}
	fmt.Println("filling shred_tests/test3.txt with test data...")
	if _, writeErr := test3.Write(test3Payload); writeErr != nil {
		panic(fmt.Sprintf("failed to write to file at shred_tests/test3.txt: %s", writeErr))
	}
	if _, seekErr := test3.Seek(0, 0); seekErr != nil {
		panic(fmt.Sprintf("failed to seek to 0 on file at shred_tests/test3.txt: %s", seekErr))
	}
	if syncErr := test3.Sync(); syncErr != nil {
		panic(fmt.Sprintf("failed to sync file at shred_tests/test3.txt: %s", syncErr))
	}
	fmt.Println("attempting to shred file at shred_tests/test1.txt...")
	progress, finalErr, initShredErr := shred(test1, false, 0, 4096, 5)
	if initShredErr != nil {
		t.Fatalf("failed to begin shredding file at shred_tests/test1.txt: %s", initShredErr)
	}
	for message := range progress {
		log(os.Stdout, message.level, logDebug, message.message)
	}
	if shredErr := <-finalErr; shredErr != nil {
		t.Fatalf("failed to shred file at shred_tests/test1.txt: %s", shredErr)
	}
	test1Read := make([]byte, len(strings.Join(test1Payload, "\n")+"\n"))
	if _, readErr := test1.ReadAt(test1Read, 0); readErr != nil {
		panic(fmt.Sprintf("failed to read file at shred_tests/test1.txt: %s", readErr))
	}
	if slices.Equal(test1Read, []byte(strings.Join(test1Payload, "\n")+"\n")) {
		t.Fatalf("file at shred_tests/test1.txt is unmodified")
	}
	if closeErr := test1.Close(); closeErr != nil {
		panic(fmt.Sprintf("failed to close file at shred_tests/test1.txt: %s", closeErr))
	}
	var byteList []string
	for _, b := range test1Read {
		byteList = append(byteList, strconv.Itoa(int(b)))
	}
	fmt.Printf("current bytes in shred_test/test1.txt in decimal: %s\n", strings.Join(byteList, " "))
	fmt.Println("attempting to shred file at shred_tests/test2.txt...")
	progress, finalErr, initShredErr = shred(test2, false, 0, 4096, 5)
	if initShredErr != nil {
		t.Fatalf("failed to begin shredding file at shred_tests/test2.txt: %s", initShredErr)
	}
	for message := range progress {
		log(os.Stdout, message.level, logDebug, message.message)
	}
	if shredErr := <-finalErr; shredErr != nil {
		t.Fatalf("failed to shred file at shred_tests/test2.txt: %s", shredErr)
	}
	if closeErr := test2.Close(); closeErr != nil {
		panic(fmt.Sprintf("failed to close file at shred_tests/test2.txt: %s", closeErr))
	}
	progress, finalErr, initShredErr = shred(test3, false, 40*(1<<20), 4096, 5)
	if initShredErr != nil {
		t.Fatalf("failed to begin shredding file at shred_tests/test3.txt: %s", initShredErr)
	}
	for message := range progress {
		log(os.Stdout, message.level, logDebug, message.message)
	}
	if shredErr := <-finalErr; shredErr != nil {
		t.Fatalf("failed to shred file at shred_tests/test3.txt: %s", shredErr)
	}
	test3Read := make([]byte, 45*(1<<20))
	if _, readErr := test3.ReadAt(test3Read, 0); readErr != nil {
		panic(fmt.Sprintf("failed to read file at shred_tests/test3.txt: %s", readErr))
	}
	test3End := make([]byte, 5*(1<<20))
	for i := range test3End {
		test3End[i] = 0xa0
	}
	if !slices.Equal(test3Read[(40*(1<<20)):], test3End) {
		t.Fatalf("shred_tests/test3.txt appears to be shredded in its entirety rather than to just 40 MiB")
	}
}
