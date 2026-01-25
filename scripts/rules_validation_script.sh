#!/bin/bash

FILE_PROTOCOLS=protocol.txt
FILE_ACTION=action.txt
FILE_RULES=../rules/manual/rules.rules
FILE_REQUIRES=required_fields.txt
FILE_TMP_SID=$(mktemp)

validate_action_and_protocol () {
	# Проверка является ли протокол поддреживаемым
	if grep -x "$1" "$2" &>/dev/null; then
		return 0
	else
		return 1
	fi
}

cut_field () {
	# Правильно нарезаем строку на поля для дальнейшей обработки
	signature=$(echo "${1%%(msg*}" | sed 's/[()]//g')
	metadata=${1##$signature}
	return 0
}

definition_sid () {
	sid=$(echo $REPLY | sed 's/[^[:alnum:]: ]//g' | grep -o 'sid:[^ ]*' | cut -d':' -f2)
	if grep -x "$sid" "$FILE_TMP_SID" &>/dev/null; then
		return 1
	else
		echo "$sid" >> "$FILE_TMP_SID"
		return 0
	fi
}

field_enumeration () {
	# Проверка signature
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

	# Проверка metadata
	IFS=';' read -a array_meta <<< "$metadata"
	count=0

	if [[ ! "$metadata" =~ ^\(.+\)$ ]]; then
		return 1
	fi

	for (( i=0; i<"${#array_meta[@]}"; i++ )); do
		local tmp_value=$(echo "${array_meta[$i]}" | sed 's/[() ]//g' | cut -d":" -f1)
		if [[ $tmp_value == 'metadata' ]]; then
			if [[ ! "${array_meta[$i]}" =~ 'mitre_technique_id' ]] ; then
				return 1
			fi
		fi
		if grep -x "$tmp_value" "$FILE_REQUIRES" &>/dev/null; then
			((count++))
		fi
	done
	if [[ $count = $(wc -l < "$FILE_REQUIRES") ]]; then
		return 0
	else
		return 1
	fi
}

> ../rules/checked/valid_rules.rules

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
		echo "$REPLY" >> ../rules/checked/valid_rules.rules
	fi

done < $FILE_RULES