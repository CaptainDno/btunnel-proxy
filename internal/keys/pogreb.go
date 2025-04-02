package keys

import (
	"crypto/rand"
	"github.com/akrylysov/pogreb"
	"go.uber.org/zap"
	rand2 "math/rand/v2"
)

type KeyPogreb struct {
	db     *pogreb.DB
	logger *zap.Logger
}

func Open(file string, logger *zap.Logger) (*KeyPogreb, error) {

	db, err := pogreb.Open(file, nil)
	if err != nil {
		return nil, err
	}

	return &KeyPogreb{
		db:     db,
		logger: logger,
	}, nil
}

func (k *KeyPogreb) Close() error {
	return k.db.Close()
}

func (k *KeyPogreb) GetKey(keyID []byte) []byte {
	res, err := k.db.Get(keyID)

	if err != nil {
		k.logger.Error("failed to retrieve key", zap.Error(err))
	}

	return res
}

func (k *KeyPogreb) SetKey(id, key []byte) error {
	return k.db.Put(id, key)
}

func (k *KeyPogreb) GetRandom() ([]byte, []byte, error) {

	target := rand2.Uint32N(k.db.Count())

	iter := k.db.Items()

	for i := uint32(0); i < target; i++ {
		_, _, _ = iter.Next()
	}

	return k.db.Items().Next()
}

func GenerateKey() ([]byte, []byte) {
	id, key := make([]byte, 20), make([]byte, 32)

	_, err := rand.Read(id)
	if err != nil {
		panic(err)
	}

	_, err = rand.Read(key)
	if err != nil {
		panic(err)
	}

	return id, key
}
