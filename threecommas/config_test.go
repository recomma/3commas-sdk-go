package threecommas

func defaultTestOptions() []ThreeCommasClientOption {
	return []ThreeCommasClientOption{
		WithAPIKey("somefakeapikey"),
		WithPrivatePEM([]byte(fakeKey)),
	}
}
