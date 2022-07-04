package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/voxtechnica/tuid-go"
)

// initTuidCmd initializes the tuid commands.
func initTuidCmd(root *cobra.Command) {
	tuidCmd := &cobra.Command{
		Use:   "tuid",
		Short: "Create new or extract embedded information from TUID(s)",
	}
	root.AddCommand(tuidCmd)

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Create the specified number of new TUIDs",
		Long:  "Create new TUIDs with the current system time and print to stdout.",
		RunE:  newTUID,
	}
	newCmd.Flags().Int16P("count", "c", 1, "Number of TUIDs to generate")
	tuidCmd.AddCommand(newCmd)

	infoCmd := &cobra.Command{
		Use:   "info <TUID> [TUID...]",
		Short: "Print TUIDInfo details for the specified TUID(s)",
		Long:  "Print TUIDInfo details for the specified TUID(s). Include a header line, if specified.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  infoTUID,
	}
	infoCmd.Flags().BoolP("header", "H", false, "Print a header row")
	tuidCmd.AddCommand(infoCmd)
}

// newTUID creates new TUID(s).
func newTUID(cmd *cobra.Command, args []string) error {
	count, err := cmd.Flags().GetInt16("count")
	if err != nil {
		return fmt.Errorf("error getting count flag: %w", err)
	}
	for i := 0; i < int(count); i++ {
		t := tuid.NewID()
		fmt.Println(t)
	}
	return nil
}

// infoTUID prints TUIDInfo details for the specified TUID(s).
func infoTUID(cmd *cobra.Command, args []string) error {
	hdr, err := cmd.Flags().GetBool("header")
	if err != nil {
		return fmt.Errorf("error getting header flag: %w", err)
	}
	if hdr {
		fmt.Println("TUID\tTimestamp\tEntropy")
	}
	for _, arg := range args {
		t, err := tuid.TUID(arg).Info()
		if err != nil {
			return fmt.Errorf("error parsing TUID %s: %w", arg, err)
		}
		fmt.Printf("%s\t%s\t%d\n", t.ID, t.Timestamp, t.Entropy)
	}
	return nil
}
