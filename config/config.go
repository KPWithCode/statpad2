package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	supa "github.com/nedpals/supabase-go"
	
)

var (
	SupabaseProjectID string
	SupabaseKey    string
	SupabaseUrl       string
	SupabaseClient *supa.Client
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	SupabaseProjectID = os.Getenv("SUPABASE_PROJECT_ID")
    SupabaseKey = os.Getenv("SUPABASE_API_KEY")
	SupabaseUrl = "https://" + SupabaseProjectID + ".supabase.co"

	SupabaseClient = supa.CreateClient(SupabaseUrl, SupabaseKey)
	
	// supabaseUrl := os.Getenv("SUPABASE_URL")	
}
