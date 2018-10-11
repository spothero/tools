package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodIntersects(t *testing.T) {
	testTime1, err := time.Parse(time.RFC3339, "2018-05-25T13:14:15Z")
	require.NoError(t, err)
	testTime2, err := time.Parse(time.RFC3339, "2018-05-26T13:14:15Z")
	require.NoError(t, err)
	p := Period{Start: testTime1, End: testTime2}
	tests := []struct {
		name           string
		expectedResult bool
		p              Period
		o              Period
	}{
		{
			"True when start intersects",
			true,
			p,
			Period{
				Start: p.Start,
				End:   p.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"True when end intersects",
			true,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(1) * time.Minute),
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"True when start and end intersects through containment",
			true,
			p,
			Period{
				Start: p.Start,
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"true when start and end contain the period",
			true,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(1) * time.Minute),
				End:   p.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"False when start and end are before",
			false,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(2) * time.Minute),
				End:   p.Start.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"False when start and end are after",
			false,
			p,
			Period{
				Start: p.End.Add(time.Duration(1) * time.Minute),
				End:   p.End.Add(time.Duration(2) * time.Minute),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.p.Intersects(test.o))
		})
	}
}

func TestPeriodContains(t *testing.T) {
	testTime1, err := time.Parse(time.RFC3339, "2018-05-25T13:14:15Z")
	require.NoError(t, err)
	testTime2, err := time.Parse(time.RFC3339, "2018-05-26T13:14:15Z")
	require.NoError(t, err)
	p := Period{Start: testTime1, End: testTime2}
	tests := []struct {
		name           string
		expectedResult bool
		p              Period
		o              Period
	}{
		{
			"False when only start intersects",
			false,
			p,
			Period{
				Start: p.Start,
				End:   p.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"False when only end intersects",
			false,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(1) * time.Minute),
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"True when start and end intersects through containment",
			true,
			p,
			Period{
				Start: p.Start,
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"False when start and end contain the period",
			false,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(1) * time.Minute),
				End:   p.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"False when start and end are before",
			false,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(2) * time.Minute),
				End:   p.Start.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"False when start and end are after",
			false,
			p,
			Period{
				Start: p.End.Add(time.Duration(1) * time.Minute),
				End:   p.End.Add(time.Duration(2) * time.Minute),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.p.Contains(test.o))
		})
	}
}

func TestPeriodContainsAny(t *testing.T) {
	start, err := time.Parse(time.RFC3339, "2018-05-24T07:14:16-06:00")
	require.NoError(t, err)
	end, err := time.Parse(time.RFC3339, "2018-05-25T07:14:14-06:00")
	require.NoError(t, err)
	p := Period{
		Start: start,
		End:   end,
	}
	pos := Period{
		Start: time.Time{},
		End:   end,
	}
	poe := Period{
		Start: start,
		End:   time.Time{},
	}
	tests := []struct {
		name           string
		expectedResult bool
		p              Period
		o              Period
	}{
		{
			"Identical time periods are contained (start)",
			true,
			p,
			p,
		}, {
			"True when start is contained",
			true,
			p,
			Period{
				Start: p.Start,
				End:   p.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"True when end is contained",
			true,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(1) * time.Minute),
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"True when period is fully contained",
			true,
			p,
			Period{
				Start: p.Start.Add(time.Duration(1) * time.Minute),
				End:   p.End.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"False when period is fully before",
			false,
			p,
			Period{
				Start: p.Start.Add(-time.Duration(2) * time.Minute),
				End:   p.Start.Add(-time.Duration(1) * time.Minute),
			},
		}, {
			"False when period is fully after",
			false,
			p,
			Period{
				Start: p.End.Add(time.Duration(1) * time.Minute),
				End:   p.End.Add(time.Duration(2) * time.Minute),
			},
		}, {
			"True when open starts period start time is before requested time",
			true,
			pos,
			Period{
				Start: pos.Start.Add(-time.Duration(1) * time.Minute),
				End:   pos.Start.AddDate(1, 0, 0),
			},
		}, {
			"False when open starts period start time is after requested time",
			false,
			pos,
			Period{
				Start: pos.End,
				End:   pos.End.Add(time.Duration(1) * time.Minute),
			},
		}, {
			"True when open ends period end time is after requested time",
			true,
			poe,
			Period{
				Start: poe.Start.Add(time.Duration(1) * time.Minute),
				End:   poe.Start.AddDate(2, 0, 0),
			},
		}, {
			"False when open ends period end time is before requested time",
			false,
			poe,
			Period{
				Start: poe.Start.AddDate(-1, 0, 0),
				End:   poe.Start.Add(-time.Duration(1) * time.Minute),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.p.ContainsAny(test.o))
		})
	}
}

func TestPeriodLess(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		p              Period
		d              time.Duration
	}{
		{
			"01/01/2018 05:00 - 01/01/2018 21:00 is less than 24 hours",
			true,
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
			time.Duration(24) * time.Hour,
		}, {
			"01/01/2018 05:00 - 01/01/2018 21:00 is not less than 16 hours",
			false,
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
			time.Duration(16) * time.Hour,
		}, {
			"01/01/2018 05:00 - 01/01/2018 21:00 is not less than 1 hour",
			false,
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
			time.Duration(1) * time.Hour,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.p.Less(test.d))
		})
	}
}

func TestPeriodContainsTime(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		p              Period
		t              time.Time
	}{
		{
			"Period 01/01/2018 05:00-21:00, request for 05:00 is contained",
			true,
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
			time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
		}, {
			"Period 01/01/2018 05:00-21:00, request for 04:59 is not contained",
			false,
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
			time.Date(2018, 1, 1, 4, 59, 0, 0, time.UTC),
		}, {
			"01/01/2018 Period 21:00 - 01/02/2018 05:00, request for 21:00 is contained",
			true,
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
			time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
		}, {
			"01/01/2018 Period 21:00 - 01/02/2018 05:00, request for 20:59 is not contained",
			false,
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
			time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.p.ContainsTime(test.t))
		})
	}
}

func TestMax(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2018-05-24T07:14:16-06:00")
	require.NoError(t, err)
	t2, err := time.Parse(time.RFC3339, "2018-05-24T07:14:16-06:00")
	require.NoError(t, err)
	tests := []struct {
		name           string
		expectedResult time.Time
		t1             time.Time
		t2             time.Time
	}{
		{
			"T1 is returned when T1 and T2 are identical",
			t1,
			t1,
			t2,
		}, {
			"T1 is returned when T1 is greater than T2",
			t1,
			t1,
			t2.Add(-time.Duration(1) * time.Minute),
		}, {
			"T2 is returned when T2 is greater than T1",
			t2,
			t1.Add(-time.Duration(1) * time.Minute),
			t2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, MaxTime(t1, t2))
		})
	}
}

func TestMin(t *testing.T) {
	t1, err := time.Parse(time.RFC3339, "2018-05-24T07:14:16-06:00")
	require.NoError(t, err)
	t2, err := time.Parse(time.RFC3339, "2018-05-24T07:14:16-06:00")
	require.NoError(t, err)
	tests := []struct {
		name           string
		expectedResult time.Time
		t1             time.Time
		t2             time.Time
	}{
		{
			"T1 is returned when T1 and T2 are identical",
			t1,
			t1,
			t2,
		}, {
			"T2 is returned when T1 is greater than T2",
			t1,
			t1.Add(time.Duration(1) * time.Minute),
			t2,
		}, {
			"T1 is returned when T2 is greater than T1",
			t2,
			t1,
			t2.Add(time.Duration(1) * time.Minute),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, MinTime(t1, t2))
		})
	}
}

func TestApplicableDaysOfWeekFromContinuous(t *testing.T) {
	continuousDayTests := []struct {
		startDay       int
		endDay         int
		expectedResult *ApplicableDays
	}{
		{0, 0, &ApplicableDays{true, false, false, false, false, false, false}},
		{0, 4, &ApplicableDays{true, true, true, true, true, false, false}},
		{5, 1, &ApplicableDays{true, true, false, false, false, true, true}},
	}
	for _, test := range continuousDayTests {
		t.Run(fmt.Sprintf("start: %d, end: %d", test.startDay, test.endDay), func(t *testing.T) {
			applicableDays := NewApplicableDays(test.startDay, test.endDay)
			assert.Equal(t, applicableDays.Monday, test.expectedResult.Monday)
			assert.Equal(t, applicableDays.Tuesday, test.expectedResult.Tuesday)
			assert.Equal(t, applicableDays.Wednesday, test.expectedResult.Wednesday)
			assert.Equal(t, applicableDays.Thursday, test.expectedResult.Thursday)
			assert.Equal(t, applicableDays.Friday, test.expectedResult.Friday)
			assert.Equal(t, applicableDays.Saturday, test.expectedResult.Saturday)
			assert.Equal(t, applicableDays.Sunday, test.expectedResult.Sunday)
		})
	}
}

func TestFloatingPeriodContiguous(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
	}{
		{
			"Floating Period 00:00-00:00 is contiguous",
			true,
			FloatingPeriod{
				Start: time.Duration(0) * time.Hour,
				End:   time.Duration(0) * time.Hour,
			},
		}, {
			"Floating Period 12:34-12:34 is contiguous",
			true,
			FloatingPeriod{
				Start: (time.Duration(12) * time.Hour) + (time.Duration(34) * time.Minute),
				End:   (time.Duration(12) * time.Hour) + (time.Duration(34) * time.Minute),
			},
		}, {
			"Floating Period 00:00-00:01 is non-contiguous",
			false,
			FloatingPeriod{
				Start: time.Duration(0) * time.Hour,
				End:   time.Duration(1) * time.Minute,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.Contiguous())
		})
	}
}

func TestFloatingPeriodAtDate(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult Period
		fp             FloatingPeriod
		d              time.Time
	}{
		{
			"Floating Period 05:00-21:00 at 11/13/2018 01:23:45 returns 11/13/2018 05:00-21:00",
			Period{
				Start: time.Date(2018, 11, 13, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 11, 13, 21, 00, 0, 0, time.UTC),
			},
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 11, 13, 1, 23, 45, 59, time.UTC),
		}, {
			`Floating Period 21:00-05:00 at 11/13/2018 01:23:45 returns
			11/13/2018 21:00 - 11/14/2018 05:00`,
			Period{
				Start: time.Date(2018, 11, 13, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 11, 14, 5, 00, 0, 0, time.UTC),
			},
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 11, 13, 1, 23, 45, 59, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.AtDate(test.d))
		})
	}
}

func TestFloatingPeriodContains(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
		p              Period
	}{
		// Test when all days are valid (starts before ends)
		{
			"Floating Period 05:00-21:00, request for 05:00-20:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 05:00-21:00 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 1/1/2018 05:00 - 1/2/2018 20:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 20, 59, 0, 0, time.UTC),
			},
		},
		// Test when all days are valid (starts AFTER ends)
		{
			"Floating Period 21:00-05:00, request for 05:00-20:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 21:00-05:00 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 1/1/2018 21:00 - 1/2/2018 04:59 is not contained",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 4, 59, 0, 0, time.UTC),
			},
		},
		// Test contiguous 24 hour periods
		{
			"Floating Period 00:00-00:00, request for 1/1/2018 05:00 - 1/2/2018 20:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(0) * time.Hour,
				End:   time.Duration(0) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 12:00-12:00, request for 1/1/2018 05:00 - 1/2/2018 20:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(12) * time.Hour,
				End:   time.Duration(12) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 23:59-23:59, request for 1/1/2018 05:00 - 1/2/2018 20:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(23)*time.Hour + time.Duration(59)*time.Minute,
				End:   time.Duration(23)*time.Hour + time.Duration(59)*time.Minute,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 23:59-23:59, request for 1/1/2018 05:00 - 1/2/2018 20:59 is not contained since Friday is not applicable",
			false,
			FloatingPeriod{
				Start: time.Duration(23)*time.Hour + time.Duration(59)*time.Minute,
				End:   time.Duration(23)*time.Hour + time.Duration(59)*time.Minute,
				Days:  NewApplicableDays(0, 3),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 5, 20, 59, 0, 0, time.UTC),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.Contains(test.p))
		})
	}
}

func TestFloatingPeriodIntersects(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
		p              Period
	}{
		// Test when all days are valid (starts before ends)
		{
			"Floating Period 05:00-21:00, request for 05:00-20:59 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 20:59-05:00 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 00:00-4:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 4, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 21:00-21:01 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 1, 0, 0, time.UTC),
			},
		},
		// Test when all days are valid (starts AFTER ends)
		{
			"Floating Period 21:00-05:00, request for 05:00-20:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 20:59-05:00 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 00:00-4:59 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 4, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 05:00-05:01 is intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 05, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 05, 1, 0, 0, time.UTC),
			},
		},
		// Test when we have gaps in days (starts before ends)
		{
			"Floating Period 05:00-21:00, request for Mon 01/01/2018 05:00 - Mon 01/01/2018 20:59 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for Sun 12/31/2017 05:00 - Sun 12/31/2017 20:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 31, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2017, 12, 31, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for Sun 12/31/2017 05:00 - Mon 01/01/2018 20:59 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 31, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for Sat 12/30/2017 20:59 - Sun 12/31/2018 00:00 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 30, 20, 59, 0, 0, time.UTC),
				End:   time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		// Test when we have gaps in days (starts AFTER ends)
		{
			"Floating Period 21:00-05:00, request for Mon 01/01/2018 05:00 - Mon 01/01/2018 20:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for Sun 12/31/2017 05:00 - Sun 12/31/2017 20:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 31, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2017, 12, 31, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for Sun 12/31/2017 05:00 - Mon 01/01/2018 20:59 is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 31, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for Sat 12/30/2017 20:59 - Sun 12/31/2018 00:00 is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 5),
			},
			Period{
				Start: time.Date(2017, 12, 30, 20, 59, 0, 0, time.UTC),
				End:   time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period Days 0-5 00:00-00:00, request for M-F at any time is intersected",
			true,
			FloatingPeriod{
				Start: time.Duration(0) * time.Hour,
				End:   time.Duration(0) * time.Hour,
				Days:  NewApplicableDays(0, 4),
			},
			Period{
				Start: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 5, 23, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period Days 0-5 00:00-00:00, request for Sa-Su at any time is not intersected",
			false,
			FloatingPeriod{
				Start: time.Duration(0) * time.Hour,
				End:   time.Duration(0) * time.Hour,
				Days:  NewApplicableDays(0, 4),
			},
			Period{
				Start: time.Date(2018, 1, 6, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 7, 23, 59, 0, 0, time.UTC),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.Intersects(test.p))
		})
	}
}

func TestFloatingPeriodContainsTime(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
		t              time.Time
	}{
		{
			"Floating Period 05:00-21:00, request for 05:00 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
		}, {
			"Floating Period 05:00-21:00, request for 04:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 1, 1, 4, 59, 0, 0, time.UTC),
		}, {
			"Floating Period 21:00-05:00, request for 21:00 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
		}, {
			"Floating Period 21:00-05:00, request for 20:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
		}, {
			"Floating Period Tu-Su 21:00-05:00, request for 01/01/2018 20:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(1, 6),
			},
			time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.ContainsTime(test.t))
		})
	}
}

func TestFloatingPeriodContainsStart(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
		p              Period
	}{
		{
			"Floating Period 05:00-21:00, request for 05:00-20:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 05:00-21:00 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 04:59-21:00 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 4, 59, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 21:00-04:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 4, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 20:59-04:59 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 4, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 21:00-05:00 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.ContainsStart(test.p))
		})
	}
}

func TestFloatingPeriodContainsEnd(t *testing.T) {
	tests := []struct {
		name           string
		expectedResult bool
		fp             FloatingPeriod
		p              Period
	}{
		{
			"Floating Period 05:00-21:00, request for 05:00-16:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 20, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 05:00-21:00, request for 05:00-21:00 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(5) * time.Hour,
				End:   time.Duration(21) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 5, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 21:00-04:59 is contained",
			true,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 4, 59, 0, 0, time.UTC),
			},
		}, {
			"Floating Period 21:00-05:00, request for 21:00-05:00 is not contained",
			false,
			FloatingPeriod{
				Start: time.Duration(21) * time.Hour,
				End:   time.Duration(5) * time.Hour,
				Days:  NewApplicableDays(0, 6),
			},
			Period{
				Start: time.Date(2018, 1, 1, 21, 0, 0, 0, time.UTC),
				End:   time.Date(2018, 1, 2, 5, 0, 0, 0, time.UTC),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedResult, test.fp.ContainsEnd(test.p))
		})
	}
}
