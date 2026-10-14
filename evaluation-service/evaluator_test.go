package main

import "testing"

func TestGetDeterministicBucketIsStableAndBounded(t *testing.T) {
	first := getDeterministicBucket("user-1flag-a")
	if first != getDeterministicBucket("user-1flag-a") {
		t.Fatal("bucket is not deterministic")
	}
	if first < 0 || first > 99 {
		t.Fatalf("bucket = %d, want value between 0 and 99", first)
	}
}

func TestRunEvaluationLogic(t *testing.T) {
	app := &App{}
	userID := "user-1"
	flagName := "flag-a"
	bucket := getDeterministicBucket(userID + flagName)

	tests := []struct {
		name string
		info *CombinedFlagInfo
		want bool
	}{
		{name: "disabled flag", info: &CombinedFlagInfo{Flag: &Flag{Name: flagName}}, want: false},
		{name: "enabled without rule", info: &CombinedFlagInfo{Flag: &Flag{Name: flagName, IsEnabled: true}}, want: true},
		{name: "disabled rule", info: &CombinedFlagInfo{Flag: &Flag{Name: flagName, IsEnabled: true}, Rule: &TargetingRule{IsEnabled: false}}, want: true},
		{name: "percentage includes bucket", info: &CombinedFlagInfo{Flag: &Flag{Name: flagName, IsEnabled: true}, Rule: &TargetingRule{IsEnabled: true, Rules: Rule{Type: "PERCENTAGE", Value: float64(bucket + 1)}}}, want: true},
		{name: "invalid percentage value", info: &CombinedFlagInfo{Flag: &Flag{Name: flagName, IsEnabled: true}, Rule: &TargetingRule{IsEnabled: true, Rules: Rule{Type: "PERCENTAGE", Value: "50"}}}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := app.runEvaluationLogic(test.info, userID); got != test.want {
				t.Fatalf("runEvaluationLogic() = %t, want %t", got, test.want)
			}
		})
	}
}
