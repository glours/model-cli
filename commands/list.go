package commands

import (
	"bytes"
	"fmt"
	"github.com/docker/go-units"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/commands/formatter"
	"github.com/docker/model-cli/desktop"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func newListCmd(desktopClient *desktop.Client) *cobra.Command {
	var jsonFormat, openai, quiet bool
	c := &cobra.Command{
		Use:     "list [OPTIONS]",
		Aliases: []string{"ls"},
		Short:   "List the available models that can be run with the Docker Model Runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			if openai && quiet {
				return fmt.Errorf("--quiet flag cannot be used with --openai flag")
			}
			models, err := listModels(openai, desktopClient, quiet, jsonFormat)
			if err != nil {
				return err
			}
			cmd.Print(models)
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	c.Flags().BoolVar(&jsonFormat, "json", false, "List models in a JSON format")
	c.Flags().BoolVar(&openai, "openai", false, "List models in an OpenAI format")
	c.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only show model IDs")
	return c
}

func listModels(openai bool, desktopClient *desktop.Client, quiet bool, jsonFormat bool) (string, error) {
	if openai {
		models, err := desktopClient.ListOpenAI()
		if err != nil {
			err = handleClientError(err, "Failed to list models")
			return "", handleNotRunningError(err)
		}
		return models, nil
	}
	models, err := desktopClient.List()
	if err != nil {
		err = handleClientError(err, "Failed to list models")
		return "", handleNotRunningError(err)
	}
	if jsonFormat {
		jsonModels, err := formatter.ToStandardJSON(models)
		if err != nil {
			return "", err
		}
		return jsonModels, nil
	}
	if quiet {
		var modelIDs string
		for _, m := range models {
			if len(m.ID) < 19 {
				fmt.Fprintf(os.Stderr, "invalid image ID for model: %v\n", m)
				continue
			}
			modelIDs += fmt.Sprintf("%s\n", m.ID[7:19])
		}
		return modelIDs, nil
	}
	return prettyPrintModels(models), nil
}

func prettyPrintModels(models []desktop.Model) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"MODEL", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "MODEL ID", "CREATED", "SIZE"})

	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT, // MODEL
		tablewriter.ALIGN_LEFT, // PARAMETERS
		tablewriter.ALIGN_LEFT, // QUANTIZATION
		tablewriter.ALIGN_LEFT, // ARCHITECTURE
		tablewriter.ALIGN_LEFT, // MODEL ID
		tablewriter.ALIGN_LEFT, // CREATED
		tablewriter.ALIGN_LEFT, // SIZE
	})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, m := range models {
		if len(m.Tags) == 0 {
			fmt.Fprintf(os.Stderr, "no tags found for model: %v\n", m)
			continue
		}
		if len(m.ID) < 19 {
			fmt.Fprintf(os.Stderr, "invalid image ID for model: %v\n", m)
			continue
		}
		table.Append([]string{
			m.Tags[0],
			m.Config.Parameters,
			m.Config.Quantization,
			m.Config.Architecture,
			m.ID[7:19],
			units.HumanDuration(time.Since(time.Unix(m.Created, 0))) + " ago",
			m.Config.Size,
		})
	}

	table.Render()
	return buf.String()
}
