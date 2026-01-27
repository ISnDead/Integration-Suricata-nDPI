#!/bin/bash

FILE_PROTOCOLS=protocol.txt
FILE_ACTION=action.txt
FILE_RULES=../rules/manual/rules.rules
FILE_REQUIRES=required_fields.txt
FILE_TMP_SID=$(mktemp)
FILE_VALID_RULES=../rules/ndpi/valid_rules.rules

validate_action_and_protocol () {
	# Checking whether the protocol is amenable
	if grep -x "$1" "$2" &>/dev/null; then
		return 0
	else
		return 1
	fi
}

cut_field () {
	# Correctly cut the string info fields for further processing
	signature=$(echo "${1%%(msg*}")
	metadata=${1##$signature}
	return 0
}

cut_metadata_field () {
	# Correctly slicing the metadata field into values for further processing
	local all_value=$(echo $1 | sed "s/, /,/g"  |cut -d":" -f2)
	IFS="," read -a stripes <<< "$all_value"
	return 0
}

definition_sid () {
	# Defining the sid of the rule
	sid=$(echo $REPLY | sed 's/[^[:alnum:]: ]//g' | grep -o 'sid:[^ ]*' | cut -d':' -f2)
	if grep -x "$sid" "$FILE_TMP_SID" &>/dev/null; then
		return 1
	else
		echo "$sid" >> "$FILE_TMP_SID"
		return 0
	fi
}

definition_requires () {
	local requires_value
	requires_value=$(printf "%s\n" "${array_meta[@]}" | grep "requires" | cut -d":" -f2)
	flag_ndpi_protocol=1
	flag_ndpi_risk=1
	cut_metadata_field "$requires_value"
	for (( i=0; i<"${#stripes[@]}"; i++ )); do
		if [[ "${stripes[$i]}" == 'keyword ndpi-protocol' ]]; then
			flag_ndpi_protocol=0
		fi
		if [[ "${stripes[$i]}" == 'keyword ndpi-risk' ]] ; then
			flag_ndpi_risk=0
         	fi
	done
}

field_enumeration () {
	# Verifying the signature part
	IFS=' ' read -a array_sign <<< "$signature"
	if [[ "${#array_sign[@]}" != '7' ]]; then
		return 1
	fi
	for (( i=0; i<"${#array_sign[@]}"; i++ )); do
		case $i in
			0)	local argument_file="$FILE_ACTION"
				;;
			1)	local argument_file="$FILE_PROTOCOLS"
				;;
			*)	continue
				;;
		esac
		if ! validate_action_and_protocol "${array_sign[$i]}" "$argument_file"; then
			return 1
		fi
	done

	# Verifying the metadata part
	if [[ ! "$metadata" =~ ^\(.+\)$ ]]; then
		return 1
	fi

	# Clean metadata field
	metadata=$(echo "$metadata" | sed 's/; /;/g')

	IFS=';' read -a array_meta <<< "$metadata"
	count=0
	local flag_mitre=1
	control_counter=$(wc -l < "$FILE_REQUIRES")

	definition_requires

	for (( i=0; i<"${#array_meta[@]}"; i++ )); do

		local tmp_value=$(echo "${array_meta[$i]}" | sed 's/[() ]//g' | cut -d":" -f1)

		# Checking the value of the metadata field
		if [[ $tmp_value == 'metadata' ]]; then
			cut_metadata_field "${array_meta[$i]}"
			for (( j=0; j<"${#stripes[@]}"; j++ )); do
				if [[ "${stripes[$j]}" =~ 'mitre_technique_id' ]] ; then
					flag_mitre=0
				fi
			done
			if [[ $flag_mitre != '0' ]]; then
				return 1
			fi
		fi

		if [[ "$tmp_value" == 'ndpi-protocol' || "$tmp_value" == 'ndpi-risk' ]]; then
			((count++))
		fi

		if grep -x "$tmp_value" "$FILE_REQUIRES" &>/dev/null; then
			((count++))
		fi
	done

	if [[ "$flag_ndpi_protocol" == '0' ]]; then
		((control_counter++))
	fi

	if [[ "$flag_ndpi_risk" == '0' ]]; then
		((control_counter++))
	fi
	if [[ "$count" == "$control_counter" ]]; then
		return 0
	else
		return 1
	fi
}

> $FILE_VALID_RULES

while read; do
	valid_string_flag=0
	[[ "$REPLY" =~ ^[[:space:]]*(#|$) ]] && continue
	definition_sid
	if [[ "$?" != '0' ]]; then
		echo "Error - double sid:$sid"
		continue
	else
		((valid_string_flag++))
	fi
	cut_field "$REPLY"
	field_enumeration
	if [[ "$?" != '0' ]]; then
		echo "Error sid:$sid"
	else
		((valid_string_flag++))
	fi
	if [[ $valid_string_flag == '2' ]]; then
		echo "$REPLY" >> $FILE_VALID_RULES
	fi
done < $FILE_RULES
