package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

type Block struct {
	Data      []byte
	PrevHash  []byte
	Hash      []byte
	Timestamp int64
}

type Blockchain struct {
	Blocks []*Block
}

func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevHash, b.Data, timestamp}, []byte{})
	hash := sha256.Sum256(headers)

	b.Hash = hash[:]
}

func (b *Blockchain) AddBlock(data string) {
	prevBlock := b.Blocks[len(b.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	b.Blocks = append(b.Blocks, newBlock)
}

func NewBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		Data:      []byte(data),
		PrevHash:  prevHash,
		Hash:      []byte{},
	}
	block.SetHash()
	return block
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}

func main() {
	b := NewBlockchain()

	b.AddBlock("Test")
	for i := range b.Blocks {
		fmt.Printf("%v: %x\n", i, b.Blocks[i].Hash)
	}
}
