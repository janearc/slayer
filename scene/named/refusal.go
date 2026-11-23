package named

// refusal is a way this package says no. It is a string constant, so no
// other code can assign to one, and errors.Is finds it by its words.
type refusal string

// Error is the refusal's words.
func (why refusal) Error() string { return string(why) }
