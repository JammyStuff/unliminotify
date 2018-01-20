package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jammystuff/gocineworld"
	"github.com/jammystuff/unliminotify/util"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// cinemasCmd represents the cinemas command
var cinemasCmd = &cobra.Command{
	Use:   "cinemas",
	Short: "List cinemas",
	Long:  `List the available cinemas`,
	Run:   run,
}

func init() {
	rootCmd.AddCommand(cinemasCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cinemasCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cinemasCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func printCinemas(listings gocineworld.Listings) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name"})

	for _, cinema := range listings.Cinemas {
		id := cinema.ID
		name := cinema.Name()
		table.Append([]string{strconv.Itoa(id), name})
	}

	table.Render()
}

func run(cmd *cobra.Command, args []string) {
	listingsXML := util.FetchListingsXML()
	listings := util.ParseListingsXML(listingsXML)
	fmt.Print("\n")
	printCinemas(listings)
}
