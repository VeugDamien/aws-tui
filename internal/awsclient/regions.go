package awsclient

// Regions returns the list of AWS regions offered in the region picker.
// A curated static list keeps the picker usable offline and avoids an extra API
// call; eu-west-3 (Paris) is listed first as the common default.
func Regions() []string {
	return []string{
		"eu-west-3",    // Paris
		"eu-west-1",    // Ireland
		"eu-west-2",    // London
		"eu-central-1", // Frankfurt
		"eu-north-1",   // Stockholm
		"eu-south-1",   // Milan
		"us-east-1",    // N. Virginia
		"us-east-2",    // Ohio
		"us-west-1",    // N. California
		"us-west-2",    // Oregon
		"ca-central-1",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-northeast-1",
		"ap-northeast-2",
		"sa-east-1",
	}
}
