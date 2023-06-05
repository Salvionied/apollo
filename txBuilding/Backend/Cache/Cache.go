package Cache

import (
	"encoding/json"
	"fmt"
	"os"
)

func Get[T any](key string, val interface{}) bool {
	dat, err := os.ReadFile(fmt.Sprintf("./tmp/%s.txt", key))
	if err != nil {
		return false
	}
	err = json.Unmarshal(dat, &val)
	if err != nil {
	}
	return true
}

func Set[T any](key string, value T) {
	val, err := json.Marshal(value)
	if err != nil {
	}
	err = os.WriteFile(fmt.Sprintf("./tmp/%s.txt", key), val, 0644)
	if err != nil {
	}

}
