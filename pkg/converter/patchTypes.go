package converter

type IngressPatch struct {
	AddAnnotations map[string]string `json:"addAnnotations"`
}
