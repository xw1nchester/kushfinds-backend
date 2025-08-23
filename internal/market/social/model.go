package social

type Social struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// TODO: посмотреть почему в модели brand ошибка, если использовать встраивание
type EntitySocial struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Url  string `json:"url"`
}
