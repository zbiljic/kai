package cmd

import (
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"
)

// addCommonLLMFlags adds the common LLM provider and model flags to a command
func addCommonLLMFlags(cmd *cobra.Command, provider *ProviderType, model *string) {
	cmd.Flags().VarP(enumflag.New(provider, "provider", ProviderIds, enumflag.EnumCaseInsensitive), "provider", "p", "LLM provider to use (phind, openai, claude, googleai, openrouter, groq, deepseek)")
	cmd.Flags().StringVarP(model, "model", "m", "", "Specific model to use for the selected provider")
}
