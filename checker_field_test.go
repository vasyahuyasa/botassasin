package main

import "testing"

func Test_fieldChecker_Check(t *testing.T) {
	tests := []struct {
		name          string
		cfg           fieldCheckerConfig
		logLine       logLine
		wantHarm      harmScore
		wantDescision instantDecision
	}{
		{
			name: "suffix whitelist",
			cfg: fieldCheckerConfig{
				FieldName: "user_agent",
				Contains:  []string{"botassasin"},
				Action:    "whitelist",
			},
			logLine: logLine{
				fields: map[string]string{
					"user_agent": "testing_botassasin",
				},
			},
			wantHarm:      0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "prefix whitelist",
			cfg: fieldCheckerConfig{
				FieldName: "user_agent",
				Contains:  []string{"testing"},
				Action:    "whitelist",
			},
			logLine: logLine{
				fields: map[string]string{
					"user_agent": "testing_botassasin",
				},
			},
			wantHarm:      0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "middle whitelist",
			cfg: fieldCheckerConfig{
				FieldName: "user_agent",
				Contains:  []string{"testing"},
				Action:    "whitelist",
			},
			logLine: logLine{
				fields: map[string]string{
					"user_agent": "unit_testing_botassasin",
				},
			},
			wantHarm:      0,
			wantDescision: decisionWhitelist,
		},
		{
			name: "no field in list",
			cfg: fieldCheckerConfig{
				FieldName: "user_agent",
				Contains:  []string{"testing"},
				Action:    "block",
			},
			logLine: logLine{
				fields: map[string]string{
					"request": "/go/unit/testing",
					"referer": "vs code",
				},
			},
			wantHarm:      0,
			wantDescision: decisionNone,
		},
		{
			name: "simple block",
			cfg: fieldCheckerConfig{
				FieldName: "user_agent",
				Contains:  []string{"harmbot"},
				Action:    "block",
			},
			logLine: logLine{
				fields: map[string]string{
					"request":    "/go/unit/testing",
					"referer":    "vs code",
					"user_agent": "go-unit-test-harmbot2000",
				},
			},
			wantHarm:      0,
			wantDescision: decisionBan,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, err := newFieldChecker(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			gotHarm, gotDescision := fc.Check(tt.logLine)
			if gotHarm != tt.wantHarm {
				t.Errorf("fieldChecker.Check() gotHarm = %v, want %v", gotHarm, tt.wantHarm)
			}
			if gotDescision != tt.wantDescision {
				t.Errorf("fieldChecker.Check() gotDescision = %v, want %v", gotDescision, tt.wantDescision)
			}
		})
	}
}
