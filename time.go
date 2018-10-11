package core

import (
	"reflect"
	"time"
)

const (
	// MsInSec is the number of milliseconds in a second
	MsInSec = 1000
	// HoursInDay is the number of hours in a single day
	HoursInDay = 24
)

// Period defines a block of time bounded by a start and end.
type Period struct {
	Start time.Time
	End   time.Time
}

// FloatingPeriod defines a period which defines a bound set of time which is applicable
// generically to any given date, but is not associated with any particular date.
type FloatingPeriod struct {
	Start time.Duration
	End   time.Duration
	Days  ApplicableDays
}

// ApplicableDays is a structure for storing what days of week something is valid for.
// This is particularly important when schedules are applicable (i.e. hours of operation &
// inventory rules)
type ApplicableDays struct {
	Monday    bool
	Tuesday   bool
	Wednesday bool
	Thursday  bool
	Friday    bool
	Saturday  bool
	Sunday    bool
}

// Intersects returns true if the other time period intersects the Period upon
// which the method was called.
func (p Period) Intersects(other Period) bool {
	// Calculate max(starts) < min(ends)
	return MaxTime(p.Start, other.Start).Before(MinTime(p.End, other.End))
}

// Contains returns true if the other time period is contained within the Period
// upon which the method was called. The time period is treated as inclusive on beginning and
// exclusive on the ends, eg [p.Start, p.End)
func (p Period) Contains(other Period) bool {
	s := (p.Start.Before(other.Start) || p.Start.Equal(other.Start)) && p.End.After(other.End)
	e := p.Start.Before(other.End) && p.End.After(other.End)
	return s && e
}

// ContainsAny returns true if the other time periods start or end  is contained within the Period
// upon which the method was called.
func (p Period) ContainsAny(other Period) bool {
	if p.Start.IsZero() {
		// If the start period is "empty" anything before our end time is contained
		return p.End.After(other.Start)
	} else if p.End.IsZero() {
		// If the end period is "empty" anything after or including our start time is contained
		return p.Start.Before(other.Start) || p.Start.Equal(other.Start)
	}
	// Otherwise, check for inclusion on start and ends times
	s := (p.Start.Before(other.Start) || p.Start.Equal(other.Start)) && p.End.After(other.Start)
	e := p.Start.Before(other.End) && p.End.After(other.End)
	return s || e
}

// Less returns true if the duration of the period is less than the supplied duration
func (p Period) Less(d time.Duration) bool {
	return p.End.Sub(p.Start) < d
}

// MaxTime returns the maximum of two timestamps, or the first timestamp if equal
func MaxTime(t1 time.Time, t2 time.Time) time.Time {
	if t2.After(t1) {
		return t2
	}
	return t1
}

// MinTime returns the minimum of two timestamps, or the first timestamp if equal
func MinTime(t1 time.Time, t2 time.Time) time.Time {
	if t2.Before(t1) {
		return t2
	}
	return t1
}

// TimeApplicable determines if the given timestamp is valid on the associated day of the week
func (ad ApplicableDays) TimeApplicable(t time.Time) bool {
	wd := t.Weekday()
	switch wd {
	case time.Sunday:
		return ad.Sunday
	case time.Monday:
		return ad.Monday
	case time.Tuesday:
		return ad.Tuesday
	case time.Wednesday:
		return ad.Wednesday
	case time.Thursday:
		return ad.Thursday
	case time.Friday:
		return ad.Friday
	case time.Saturday:
		return ad.Saturday
	default:
		return false
	}
}

// NewApplicableDays translates continuous days of week to a struct with bools representing each
// day of the week. Note that this implementation is dependent on the ordering
// of days of the week in the applicableDaysOfWeek struct. It *must* match
// the way Django handles days of the week i.e. Monday=0, Sunday=6, otherwise
// the days of the week will be wrong.
// See https://github.com/django/django/blob/e4e44b92ddf717e34401c0bd1a0ad203a6b3e132/django/utils/dates.py#L5
func NewApplicableDays(startDay int, endDay int) ApplicableDays {
	applicableDays := &ApplicableDays{}
	v := reflect.ValueOf(applicableDays).Elem()
	for i := 0; i < 7; i++ {
		var dayApplicable bool
		if startDay <= endDay {
			dayApplicable = startDay <= i && endDay >= i
		} else {
			dayApplicable = startDay <= i || endDay >= i
		}
		v.Field(i).SetBool(dayApplicable)
	}
	return *applicableDays
}

// Contiguous returns true if starts time is equal to end time. It does not consider applicable
// days.
func (fp FloatingPeriod) Contiguous() bool {
	return fp.Start == fp.End
}

// AtDate returns the Floating Period offset relative to midnight of the date provided
func (fp FloatingPeriod) AtDate(date time.Time) Period {
	midnight := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	offsetDate := Period{Start: midnight.Add(fp.Start), End: midnight.Add(fp.End)}
	if fp.Start > fp.End {
		offsetDate.End = offsetDate.End.AddDate(0, 0, 1)
	}
	return offsetDate
}

// Contains determines if the FloatingPeriod contains the specified Period. Because the starts
// and ends time may not align with our calculated applied floating period, the function scans from
// period day start - 1 to period day + 1 to ensure that all possible overlaps are accounted for.
func (fp FloatingPeriod) Contains(period Period) bool {
	// If the floating period has no gaps, simply verify that the rule is applicable for every
	// day between the period start and end
	if fp.Contiguous() {
		currDay := period.Start
		for currDay.Before(period.End) || currDay.Equal(period.End) {
			if !fp.Days.TimeApplicable(currDay) {
				return false
			}
			currDay = currDay.AddDate(0, 0, 1)
		}
		return true
	}

	// Otherwise adjust the Floating Period to the requested period and check applicability
	currDate := fp.AtDate(period.Start)
	if !fp.Days.TimeApplicable(currDate.Start) || !currDate.Contains(period) {
		return false
	}
	return true
}

// ContainsTime determines if the Period contains the specified time.
func (p Period) ContainsTime(t time.Time) bool {
	return (p.Start.Before(t) || p.Start.Equal(t)) && p.End.After(t)
}

// ContainsTime determines if the FloatingPeriod contains the specified time.
func (fp FloatingPeriod) ContainsTime(t time.Time) bool {
	if !fp.Days.TimeApplicable(t) {
		return false
	}
	return fp.AtDate(t).ContainsTime(t)
}

// Intersects determines if the FloatingPeriod intersects the specified Period. Because the starts
// and ends time may not align with our calculated applied floating period, the function scans from
// period day start - 1 to period day + 1 to ensure that all possible overlaps are accounted for.
// If the start and ends times are equal, the method simply checks that for any given period, at
// least one day in that period occurs during this floating period.
func (fp FloatingPeriod) Intersects(period Period) bool {
	if fp.Start == fp.End {
		currDate := period.Start
		for !currDate.After(period.End) {
			if fp.Days.TimeApplicable(currDate) {
				return true
			}
			currDate = currDate.AddDate(0, 0, 1)
		}
	} else {
		currDate := fp.AtDate(period.Start.AddDate(0, 0, -1))
		dayAfterEnd := period.End.AddDate(0, 0, 1)
		// If start equals ends, then we only need to check if the date is applicable, not the times.
		completePeriod := fp.Start == fp.End
		for {
			if fp.Days.TimeApplicable(currDate.Start) && (completePeriod || currDate.Intersects(period)) {
				return true
			}
			currDate.Start = currDate.Start.AddDate(0, 0, 1)
			currDate.End = currDate.End.AddDate(0, 0, 1)
			if currDate.End.After(dayAfterEnd) {
				break
			}
		}
	}
	return false
}

// ContainsStart determines if the FloatingPeriod contains the start of a given period. Note that
// this function is a convenience function is equivalent to `fp.ContainsTime(period.Start)`.
func (fp FloatingPeriod) ContainsStart(period Period) bool {
	return fp.ContainsTime(period.Start)
}

// ContainsEnd determines if the FloatingPeriod contains the end of a given period
func (fp FloatingPeriod) ContainsEnd(period Period) bool {
	offsetHours := fp.AtDate(period.End)
	midnightEnd := time.Date(period.End.Year(), period.End.Month(), period.End.Day(), 0, 0, 0, 0, period.End.Location())

	if fp.Start > fp.End {
		// If this is an overnight rule and the rental ends during the overnight period, we want to
		// know if the day before the end day is applicable.
		if period.End.Before(offsetHours.End) || period.End.Equal(offsetHours.End) {
			if !fp.Days.TimeApplicable(period.End.AddDate(0, 0, -1)) {
				return false
			}
			return fp.End > period.End.Sub(midnightEnd)
		}
	}

	// Else, if this an overnight rule ending during the second portion of the end day, we want
	// know if the rule is applicable for that day.
	if !fp.Days.TimeApplicable(period.End) {
		return false
	}

	// Otherwise this is a normal rule, simply check if the rule is within bounds
	return offsetHours.End.After(period.End)
}
