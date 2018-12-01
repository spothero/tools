package core

import (
	"fmt"
	"reflect"
	"time"
)

const (
	// MsInSec is the number of milliseconds in a second
	MsInSec = 1000
	// HoursInDay is the number of hours in a single day
	HoursInDay = 24
	// DaysInWeek is the number of days in a week
	DaysInWeek = 7
)

// Period defines a block of time bounded by a start and end.
type Period struct {
	Start time.Time
	End   time.Time
}

// FloatingPeriod defines a period which defines a bound set of time which is applicable
// generically to any given date in a given location, but is not associated with any particular date.
type FloatingPeriod struct {
	Start    time.Duration
	End      time.Duration
	Days     ApplicableDays
	Location *time.Location
}

// ContinuousPeriod defines a period which defines a block of time in a given location, bounded by a week, which may
// span multiple days.
type ContinuousPeriod struct {
	Start    time.Duration
	End      time.Duration
	StartDOW time.Weekday
	EndDOW   time.Weekday
	Location *time.Location
}

// RecurringPeriod defines an interface for converting periods that represent abstract points in time
// into concrete periods
type RecurringPeriod interface {
	AtDate(date time.Time) Period
	FromTime(t time.Time) *Period
	Contains(period Period) bool
	ContainsTime(t time.Time) bool
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

// NewPeriod constructs a new time period from start and end times in a given location
func NewPeriod(start, end time.Time) Period {
	return Period{
		Start: start,
		End:   end,
	}
}

// NewFloatingPeriod constructs a new floating period
func NewFloatingPeriod(start, end time.Duration, days ApplicableDays, location *time.Location) FloatingPeriod {
	l := location
	if location == nil {
		l = time.UTC
	}
	return FloatingPeriod{
		Start:    start,
		End:      end,
		Days:     days,
		Location: l,
	}
}

// NewContinuousPeriod constructs a new continuous period
func NewContinuousPeriod(start, end time.Duration, startDow, endDow time.Weekday, location *time.Location) ContinuousPeriod {
	l := location
	if location == nil {
		l = time.UTC
	}
	return ContinuousPeriod{
		Start:    start,
		End:      end,
		StartDOW: startDow,
		EndDOW:   endDow,
		Location: l,
	}
}

// Intersects returns true if the other time period intersects the Period upon
// which the method was called.
func (p Period) Intersects(other Period) bool {
	// Calculate max(starts) < min(ends)
	return MaxTime(p.Start, other.Start).Before(MinTime(p.End, other.End))
}

// Contains returns true if the other time period is contained within the Period
// upon which the method was called. The time period is treated as inclusive on both ends
// eg [p.Start, p.End]
func (p Period) Contains(other Period) bool {
	s := p.Start.Before(other.Start) || p.Start.Equal(other.Start)
	e := p.End.After(other.End) || p.End.Equal(other.End)
	if !p.Start.IsZero() && !p.End.IsZero() {
		return s && e
	} else if p.Start.IsZero() && !p.End.IsZero() {
		return e
	} else if !p.Start.IsZero() && p.End.IsZero() {
		return s
	}
	return true
}

// ContainsAny returns true if the other time periods start or end is contained within the Period
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

// MonStartToSunStart normalizes Monday Start Day of Week (Mon=0, Sun=6) to Sunday Start of Week (Sun=0, Sat=6)
func MonStartToSunStart(dow int) (time.Weekday, error) {
	switch dow {
	case 0:
		return time.Monday, nil
	case 1:
		return time.Tuesday, nil
	case 2:
		return time.Wednesday, nil
	case 3:
		return time.Thursday, nil
	case 4:
		return time.Friday, nil
	case 5:
		return time.Saturday, nil
	case 6:
		return time.Sunday, nil
	}
	return time.Sunday, fmt.Errorf("unknown day of week")
}

// TimeApplicable determines if the given timestamp is valid on the associated day of the week in a given timezone
func (ad ApplicableDays) TimeApplicable(t time.Time, location *time.Location) bool {
	wd := t.In(location).Weekday()
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

// NewApplicableDaysMonStart translates continuous days of week to a struct with bools representing each
// day of the week. Note that this implementation is dependent on the ordering
// of days of the week in the applicableDaysOfWeek struct. Monday is 0, Sunday is 6.
func NewApplicableDaysMonStart(startDay int, endDay int) ApplicableDays {
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
	dateInLoc := date.In(fp.Location)
	midnight := time.Date(dateInLoc.Year(), dateInLoc.Month(), dateInLoc.Day(), 0, 0, 0, 0, fp.Location)
	offsetDate := Period{Start: midnight.Add(fp.Start), End: midnight.Add(fp.End)}
	if fp.Start >= fp.End {
		if dateInLoc.After(offsetDate.Start) || dateInLoc.Equal(offsetDate.Start) {
			offsetDate.End = offsetDate.End.AddDate(0, 0, 1)
		} else {
			offsetDate.Start = offsetDate.Start.AddDate(0, 0, -1)
		}
	}
	return offsetDate
}

// AtDate returns the ContinuousPeriod offset around the given date
func (cp ContinuousPeriod) AtDate(d time.Time) Period {
	var offsetDate Period
	var startDay time.Time
	if cp.StartDOW > d.Weekday() {
		offset := time.Duration(HoursInDay*(d.Weekday()+(DaysInWeek-cp.StartDOW))) * time.Hour
		startDay = d.Add(-offset)
		offsetDate.Start = time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, cp.Location)
		offsetDate.Start = offsetDate.Start.Add(cp.Start)
	} else {
		offset := time.Duration(HoursInDay*(d.Weekday()-cp.StartDOW)) * time.Hour
		startDay = d.Add(-offset)
		offsetDate.Start = time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, cp.Location)
		offsetDate.Start = offsetDate.Start.Add(cp.Start)
	}

	if cp.EndDOW > cp.StartDOW {
		endDay := startDay.Add(time.Duration(HoursInDay*(cp.EndDOW-cp.StartDOW)) * time.Hour)
		offsetDate.End = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, cp.Location)
		offsetDate.End = offsetDate.End.Add(cp.End)
	} else if cp.EndDOW < cp.StartDOW {
		endDay := startDay.Add(time.Duration(HoursInDay*((DaysInWeek-cp.StartDOW)+cp.EndDOW)) * time.Hour)
		offsetDate.End = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, cp.Location)
		offsetDate.End = offsetDate.End.Add(cp.End)
	} else {
		endDay := startDay
		if cp.Start >= cp.End {
			endDay = endDay.Add(time.Duration(DaysInWeek*HoursInDay) * time.Hour)
		}
		offsetDate.End = time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, cp.Location)
		offsetDate.End = offsetDate.End.Add(cp.End)
	}
	return offsetDate
}

// FromTime returns a period that extends from a given start time to the end of the floating period, or nil
// if the start time does not fall within the floating period
func (fp FloatingPeriod) FromTime(t time.Time) *Period {
	p := fp.AtDate(t)
	if !fp.Days.TimeApplicable(p.Start, fp.Location) {
		return nil
	}
	if !p.ContainsTime(t) {
		return nil
	}
	fromPeriod := NewPeriod(t, p.End)
	return &fromPeriod
}

// FromTime returns a period that extends from a given start time to the end of the continuous period, or nil
// if the start time does not fall within the continuous period
func (cp ContinuousPeriod) FromTime(t time.Time) *Period {
	p := cp.AtDate(t)
	if !p.ContainsTime(t) {
		return nil
	}
	fromPeriod := NewPeriod(t, p.End)
	return &fromPeriod
}

// Contains determines if the ContinuousPeriod contains the specified Period.
func (cp ContinuousPeriod) Contains(period Period) bool {
	return cp.AtDate(period.Start).Contains(period)
}

// Contains determines if the FloatingPeriod contains the specified Period.
func (fp FloatingPeriod) Contains(period Period) bool {
	atDate := fp.AtDate(period.Start)
	return fp.Days.TimeApplicable(atDate.Start, fp.Location) && atDate.Contains(period)
}

// ContainsTime determines if the Period contains the specified time.
func (p Period) ContainsTime(t time.Time) bool {
	if p.Start.IsZero() && p.End.IsZero() {
		return true
	} else if !p.Start.IsZero() && p.End.IsZero() {
		return p.Start.Before(t) || p.Start.Equal(t)
	} else if p.Start.IsZero() && !p.End.IsZero() {
		return p.End.After(t) || p.End.Equal(t)
	}
	return (p.Start.Before(t) || p.Start.Equal(t)) && p.End.After(t)
}

// ContainsTime determines if the FloatingPeriod contains the specified time.
func (fp FloatingPeriod) ContainsTime(t time.Time) bool {
	if !fp.Days.TimeApplicable(t, fp.Location) {
		return false
	}
	return fp.AtDate(t).ContainsTime(t)
}

// ContainsTime determines if the continuous period contains the specified time.
func (cp ContinuousPeriod) ContainsTime(t time.Time) bool {
	return cp.AtDate(t).ContainsTime(t)
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
			if fp.Days.TimeApplicable(currDate, fp.Location) {
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
			if fp.Days.TimeApplicable(currDate.Start, fp.Location) && (completePeriod || currDate.Intersects(period)) {
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
// this function is a convenience function is equivalent to `fp.containsTime(period.Start)`.
func (fp FloatingPeriod) ContainsStart(period Period) bool {
	return fp.ContainsTime(period.Start)
}

// ContainsEnd determines if the FloatingPeriod contains the end of a given period
func (fp FloatingPeriod) ContainsEnd(period Period) bool {
	offsetHours := fp.AtDate(period.End)
	midnightEnd := time.Date(period.End.Year(), period.End.Month(), period.End.Day(), 0, 0, 0, 0, fp.Location)

	if fp.Start > fp.End {
		// If this is an overnight rule and the rental ends during the overnight period, we want to
		// know if the day before the end day is applicable.
		if period.End.Before(offsetHours.End) || period.End.Equal(offsetHours.End) {
			if !fp.Days.TimeApplicable(period.End.AddDate(0, 0, -1), fp.Location) {
				return false
			}
			return fp.End > period.End.Sub(midnightEnd)
		}
	}

	// Else, if this an overnight rule ending during the second portion of the end day, we want
	// know if the rule is applicable for that day.
	if !fp.Days.TimeApplicable(period.End, fp.Location) {
		return false
	}

	// Otherwise this is a normal rule, simply check if the rule is within bounds
	return offsetHours.End.After(period.End)
}
