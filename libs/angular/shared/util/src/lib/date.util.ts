export function dateFilterComparator(
  filterLocalDateAtMidnight: Date,
  cellValue: string,
) {
  const cellDate = new Date(cellValue);
  if (filterLocalDateAtMidnight.getTime() === cellDate.getTime()) return 0;
  if (cellDate < filterLocalDateAtMidnight) return -1;
  if (cellDate > filterLocalDateAtMidnight) return 1;
  return 0;
}

export function dateSortComparator(stringDate1: string, stringDate2: string) {
  const date1 = new Date(stringDate1);
  const date2 = new Date(stringDate2);
  if (date1 === null && date2 === null) return 0;
  if (date1 === null) return -1;
  if (date2 === null) return 1;
  return date1.getTime() - date2.getTime();
}
