package elevation

// AutoApprovePrompter always returns (true, nil). Used for --yes / scripted
// runs where interactive confirmation is undesirable. The banner is still
// printed upstream, so the run log records what was auto-approved.
type AutoApprovePrompter struct{}

// Confirm implements Prompter.
func (AutoApprovePrompter) Confirm(_ string) (bool, error) { return true, nil }
