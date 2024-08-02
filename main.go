package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/labstack/echo/v4"
)

// For testing purposes it's set to 20, it should be at least 24
const (
	targetBits   = 20
	dbFile       = "db"
	blocksBucket = "blockchain"
)

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
		log.Printf("Can not serialize this block, error: %v", err)
	}

	return result.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	if err != nil {
		log.Printf("Can not deserialize this block, error: %v", err)
	}

	return &block
}

func (bc *Blockchain) AddBlock(data []byte) {

	newBlock := Deserialize(data)

	err := bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Fatalf("Could not put new block into database: %v", err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		bc.Tip = newBlock.Hash

		return nil
	})
	if err != nil {
		log.Fatalf("Could not open a database: %v", err)
	}

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

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		var data string
		bci := bc.Iterator()

		for {
			block := bci.Next()

			data += fmt.Sprintf("Prev. hash: %x\n", block.PrevHash)
			data += fmt.Sprintf("Data: %s\n", block.Data)
			data += fmt.Sprintf("Hash: %x\n", block.Hash)

			pow := NewPoW(block)
			data += fmt.Sprintf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
			data += "\n"

			if len(block.PrevHash) == 0 {
				break
			}
		}
		return c.String(http.StatusOK, data)
	})

	e.POST("/addblock", func(c echo.Context) error {
		data := c.FormValue("data")
		if data == "req" {
			return c.String(http.StatusOK, fmt.Sprintf("%x", bc.Tip))
		} else {
			bc.AddBlock([]byte(data))
			return c.String(http.StatusOK, "Block has been added")
		}
	})

	e.Logger.Fatal(e.Start(":8080"))
}
