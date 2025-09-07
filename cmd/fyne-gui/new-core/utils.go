// Путь: cmd/fyne-gui/new-core/utils.go

package newcore

import (
	"crypto/sha256"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// CreateContentID создает правильный CIDv1 из строки.
// Эта функция будет использоваться и Core, и GUI для консистентного
// вычисления идентификаторов контента.
func CreateContentID(data string) (string, error) {
	// 1. Хэшируем данные
	hash := sha256.Sum256([]byte(data))

	// 2. Создаем multihash
	mh, err := multihash.Encode(hash[:], multihash.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("ошибка создания multihash: %w", err)
	}

	// 3. Создаем CIDv1 с кодеком raw
	// cid.Raw - это стандарт для указания на сырые бинарные данные
	cidV1 := cid.NewCidV1(cid.Raw, mh)

	return cidV1.String(), nil
}
