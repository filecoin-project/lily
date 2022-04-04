package v1

// Schema version 1 adds gap find function

func init() {
	patches.Register(
		6,
		`
CREATE FUNCTION {{ .SchemaName | default "public"}}.gap_find(
	tasks       	text[],
	min_height		bigint,
	max_height		bigint,
	null_status		text,
	ok_status 		text
) RETURNS TABLE (
	height			bigint,
	task			text
) LANGUAGE plpgsql STABLE PARALLEL SAFE STRICT
AS $$
BEGIN
	RETURN QUERY WITH

	interesting_tasks AS (
		SELECT * FROM unnest(tasks) AS x(task)
	),

	all_heights_and_tasks_in_range AS (
		SELECT a.height, a.task FROM (
			(SELECT * FROM generate_series(min_height, max_height) AS x(height)) h
			CROSS JOIN
			(SELECT * FROM interesting_tasks) t
		) AS a
	),

	heights_in_processing_report AS (
		SELECT v.height, v.task, v.status, v.status_information
		FROM visor_processing_reports v
		WHERE v.height BETWEEN min_height AND max_height
	),

    complete_epochs_tasks AS (
		SELECT v.height, v.task
		FROM heights_in_processing_report v
		LEFT JOIN visor_processing_reports x
		ON v.height = x.height AND v.task = x.task AND v.status = x.status
		WHERE v.status = ok_status
		AND v.task IN (SELECT * FROM interesting_tasks)
		GROUP BY 1, 2, x.status
	),

	null_round AS (
		SELECT pr.height, t.task
		FROM heights_in_processing_report pr
		CROSS JOIN interesting_tasks t
		WHERE (
			pr.status_information = null_status OR pr.status_information IS NOT null
		)
		GROUP BY 1, 2
	)

	SELECT xor.height, xor.task FROM
	(
		(SELECT * FROM all_heights_and_tasks_in_range)
		EXCEPT
		(SELECT * FROM complete_epochs_tasks)
		EXCEPT
		(SELECT * FROM null_round)
	) AS xor
	ORDER BY xor.height DESC, xor.task;
END;$$
`)
}
