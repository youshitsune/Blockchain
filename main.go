package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

// For testing purposes it's set to 20, it should be at least 24
const (
	targetBits   = 20
	dbFile       = "db"
	blocksBucket = "blockchain"
)

type PoW struct {
	Block  *Block
	Target *big.Int
}

type Block struct {
	Data      []byte
	PrevHash  []byte
	Hash      []byte
	Timestamp int64
	Nonce     int
}

type Blockchain struct {
	Tip []byte
	DB  *bolt.DB
}

type BlockchainIterator struct {
	DB          *bolt.DB
	CurrentHash []byte
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer

	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		fmt.Printf("Can not serialize this block, error: %v", err)
	}

	return result.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	if err != nil {
		fmt.Printf("Can not deserialize this block, error: %v", err)
	}

	return &block
}

func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	err := bc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Fatalf("Could not open a database: %v", err)
	}

	newBlock := NewBlock(data, lastHash)

	err = bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Fatalf("Could not put new block into database: %v", err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		bc.Tip = newBlock.Hash

		return nil
	})
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{
		DB:          bc.DB,
		CurrentHash: bc.Tip,
	}
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.CurrentHash)
		block = Deserialize(encodedBlock)

		return nil
	})

	if err != nil {
		log.Fatalf("Could not open a database: %v", err)
	}

	i.CurrentHash = block.PrevHash

	return block
}

func (pow *PoW) PrepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.Block.PrevHash,
		pow.Block.Data,
		IntToHex(pow.Block.Timestamp),
		IntToHex(int64(targetBits)),
		IntToHex(int64(nonce)),
	}, []byte{})

	return data
}

func (pow *PoW) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining the block containing \"%s\"\n", pow.Block.Data)
	for nonce < math.MaxInt64 {
		data := pow.PrepareData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Printf("\n\n")

	return nonce, hash[:]
}

func (pow *PoW) Validate() bool {
	var hashInt big.Int

	data := pow.PrepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(pow.Target) == -1
}

func IntToHex(x int64) []byte {
	return []byte(strconv.FormatInt(x, 16))
}

func NewPoW(b *Block) *PoW {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	return &PoW{
		Block:  b,
		Target: target,
	}
}

func NewBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		Data:      []byte(data),
		PrevHash:  prevHash,
		Hash:      []byte{},
	}
	pow := NewPoW(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatalf("Error while opening the database: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Fatalf("Error while creating a bucket: %v", err)
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			err = b.Put([]byte("l"), genesis.Hash)
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	b := Blockchain{
		Tip: tip,
		DB:  db,
	}

	return &b
}

func main() {
	bc := NewBlockchain()
	defer bc.DB.Close()

	cli := CLI{bc}
	cli.Run()
}
