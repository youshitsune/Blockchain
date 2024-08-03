package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/labstack/echo/v4"
)

// For testing purposes it's set to 20, it should be at least 24
const (
	targetBits   = 20
	dbFile       = "db"
	blocksBucket = "blockchain"
)

type Blockchain struct {
	Tip []byte
	DB  *bolt.DB
}

type BlockchainIterator struct {
	DB          *bolt.DB
	CurrentHash []byte
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

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", "", []byte{})
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
		var data []string
		bci := bc.Iterator()

		for {
			block := bci.Next()

			data = append(data, string(block.Serialize()))

			if len(block.PrevHash) == 0 {
				break
			}
		}
		return c.String(http.StatusOK, strings.Join(data, "|/"))
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

	e.POST("/getdata", func(c echo.Context) error {
		var res []string
		hash := c.FormValue("data")
		bci := bc.Iterator()

		for {
			block := bci.Next()

			if hash == fmt.Sprintf("%x", block.Hash) {
				res = append(res, string(block.Data), string(block.Name))
				return c.String(http.StatusOK, strings.Join(res, "|/"))
			}

			if len(block.PrevHash) == 0 {
				break
			}
		}

		return c.String(http.StatusOK, "That block does not exist")
	})

	e.Logger.Fatal(e.Start(":8080"))
}
