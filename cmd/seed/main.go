package main

import (
	"fmt"
	"log"

	"github.com/JeremyProffitt/doc-manager/internal/config"
)

func main() {
	cfg := config.Load()
	_ = cfg

	// TODO: Initialize DynamoDB client
	// TODO: Seed user (proffitt.jeremy@gmail.com)
	// TODO: Seed customers
	// TODO: Seed default field settings

	fmt.Println("Seed script placeholder — will be implemented in Step 2")
	log.Println("Config loaded:", cfg.UsersTable, cfg.CustomersTable)
}
