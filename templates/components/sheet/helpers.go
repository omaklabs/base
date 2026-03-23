package sheet

import (
	"crypto/rand"
	"fmt"
)

func randomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("sht-%x", b)
}
