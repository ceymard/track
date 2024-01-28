
---
CREATE TABLE IF NOT EXISTS track(
	ts INT NOT NULL,
	dur INT NOT NULL,
	name text
);

CREATE INDEX IF NOT EXISTS track_by_ts ON track(ts);

--- Some quick name handling as well has hour rounding
DROP VIEW IF EXISTS track_full;
CREATE VIEW track_full AS
SELECT
  (ts - ts % (3600000)) as hr,
  ts,
  CASE
    WHEN name REGEXP '^\d+$|__i3_scratch' THEN '∅'
	ELSE replace(name, ' (Workspace)', '')
  END as nm,
  name,
  dur
FROM track;


---
DROP VIEW IF EXISTS const;
CREATE VIEW const AS
SELECT
    unixepoch(CURRENT_TIMESTAMP) * 1000
	AS ts_now,

	CASE
		when time(CURRENT_TIMESTAMP, 'localtime') < '06:00:00' then date(CURRENT_TIMESTAMP, '-07:00:00', 'localtime')
		else date(CURRENT_TIMESTAMP, 'localtime')
	end
	as start_of_today,

	unixepoch(case
		when time(CURRENT_TIMESTAMP, 'localtime') < '06:00:00' then date(CURRENT_TIMESTAMP, '-07:00:00', 'localtime')
		else date(CURRENT_TIMESTAMP, 'localtime')
	end, 'localtime') * 1000
	as ts_start_of_today;


---
DROP VIEW IF EXISTS today;
CREATE VIEW today AS
SELECT
	nm,
	sum(dur) as dur,
	(sum(dur)*100 / (SELECT sum(dur) FROM track_full t WHERE t.ts >= const.ts_start_of_today AND nm <> '__idle')) AS pct
FROM track_full t, const
WHERE t.ts >= const.ts_start_of_today
  AND nm <> '__idle'
GROUP BY nm
ORDER BY sum(dur) DESC;


---
DROP VIEW IF EXISTS today_by_hour;
CREATE VIEW today_by_hour as
	select
		strftime('%H', hr / 1000, 'unixepoch', 'localtime') as hour,
		nm,
		sum(dur) as dur
	from track_full t, const
	where t.ts >= const.ts_start_of_today
	group by hr, nm
	order by hr, sum(dur) desc;


---
DROP VIEW IF EXISTS genmon_today_by_hour;
CREATE VIEW genmon_today_by_hour as
SELECT
	'<span size="large" weight="bold">' || hour || 'h</span> '
	|| group_concat(
	'  <span style="italic"'
	|| case when nm = '∅' then ' color="#e91e63"' else '' end
	|| ' >'
	|| nm
	|| '</span> <span size="small" weight="bold" >'
	|| substr(time(dur/1000, 'unixepoch'), 4)
	|| '</span>', ', ')
	AS line
FROM today_by_hour
WHERE nm <> '__idle' -- do not show idle time
GROUP BY hour;


---
DROP VIEW IF EXISTS genmon_today;
CREATE VIEW genmon_today AS
SELECT
	'<span weight="bold">' || pct || '%</span> - '
  || '<span' ||
    CASE
		WHEN nm = '∅' THEN ' color="#e91e63" weight="bold"'
		ELSE ' color="#8bc34a" weight="bold"'
	END
 || '>' || nm || '</span>'
 || ' <span weight="bold" size="small">' || time(dur/1000, 'unixepoch') || '</span>'
 AS line
FROM today;


---
DROP VIEW IF EXISTS today_overview;
CREATE VIEW today_overview AS
SELECT
  sum(dur) FILTER (WHERE nm <> '∅') as work,
  sum(dur) FILTER (WHERE nm = '∅') as glande,
  sum(dur) FILTER (WHERE nm <> '∅') * 100 / sum(dur) as work_pct,
  sum(dur) FILTER (WHERE nm = '∅') * 100 / sum(dur) as glande_pct
FROM today;


---
DROP VIEW IF EXISTS genmon;
CREATE VIEW genmon as
SELECT '<txt>'
  || '<span' || CASE
	WHEN t.work_pct >= 50 THEN ' weight="bold" color="#8bc34a"'
	  ELSE ''
	END
  || '>' || time(t.work / 1000, 'unixepoch') || ' <span size="x-small">(' || t.work_pct || '%)</span></span> vs. '
  || '<span' || CASE
	WHEN t.glande_pct >= 50 THEN ' weight="bold" color="#e91e63"'
	  ELSE ''
	END
    ||'>∅ ' || time(t.glande / 1000, 'unixepoch') || ' <span size="x-small">(' || t.glande_pct || '%)</span></span>'
	|| '</txt>' || char(10)
	|| '<tool>'
	|| (SELECT group_concat(line, char(10)) FROM genmon_today_by_hour)
	|| char(10) || char(10)
	|| '<span size="large" weight="bold">TODAY</span>' || char(10)
	|| (SELECT group_concat(line, char(10)) FROM genmon_today)
	|| '</tool>'
AS res
FROM today_overview t;