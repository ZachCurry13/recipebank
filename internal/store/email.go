package store

// The mail server RecipeBank emails recipes and the shopping list through.
const (
	KeySMTPHost     = "smtp_host"
	KeySMTPPort     = "smtp_port"
	KeySMTPUser     = "smtp_username"
	KeySMTPPassword = "smtp_password" // a secret: never sent to the browser
	KeySMTPFrom     = "smtp_from"
)

func init() {
	SecretKeys[KeySMTPPassword] = true
	Defaults[KeySMTPPort] = "587"
}
