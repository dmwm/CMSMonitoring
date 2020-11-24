for $i in parallelize(1 to 1000)
where $i mod 9 eq 0
return $i
