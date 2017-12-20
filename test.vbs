'on error resume next

'call forceCscript

wscript.echo "in nested app"

' Store the arguments in a variable:
Set objArgs = Wscript.Arguments

' print out any args
if objArgs.Count > 0 then
	for i = 0 to objArgs.Count
		wscript.echo "Arg " & i & " = " & objArgs(i)

	next
end if


debugging = true
Randomize timer

if debugging then
	wscript.echo "TAGUPDATE: awesomeTag = 456"
	wscript.echo "TAGUPDATE: awesomeTag = ***"
	wscript.echo "TAGUPDATE: testBit = False"

	wscript.sleep(3000)
end if


i = 0
' print out all stdin
' exit when exit is sent to stdin
do
	wscript.sleep(250)


	if not debugging then
		
		UserInput = WScript.StdIn.ReadLine
		if UserInput = "exit" then break

		' build in some simulation stuff
		if UserInput = "plc" then 
			wscript.echo "TAGUPDATE: testTag = 123"
		else

			' if not a specific tag just echo out what came in
			wscript.echo UserInput
		end if

	else
		' debug mode, update random tags
		i = i + 1
		
		if i > 4 then ' check every n 1/4 seconds
			
			max=100
			min=1

			roll = (Int((max-min+1)*Rnd+min))			

			if ( roll > 70 ) then
				call updateRandomTag
			end if
			i = 0
		end if
	end if

loop



sub updateRandomTag()

	tagName = "randomTag"
	tagValue = 7 + i

	max=100
	min=1

	roll = (Int((max-min+1)*Rnd+min))			

	if ( roll > 70 ) then tagName = "sensor_1"
	if ( roll mod 5 = 0 ) then tagValue = tagValue + 5
	if ( roll < 70 and roll >5 ) then tagName = "temp"
	if ( roll < 70 ) then tagName = "sensor_2"
	if ( roll > 50 ) then tagValue = tagValue + 17
	if ( roll < 50 ) then tagValue = tagValue - 2
	if ( roll < 20 ) then tagName = "units"
	if ( roll mod 4 = 0 and roll < 90 ) then tagName = "flow"
	if ( roll > 20 and roll < 50 ) then tagName = "dollars"
	if ( roll > 60 and roll < 69 ) then tagName = "sound_level"
	if ( roll mod 8 = 0 ) then tagName = "awesomeTag"

	tagValue = tagValue + roll

	wscript.echo "TAGUPDATE: " & tagName & " = " & tagValue


	if roll mod 2 = 0 then 
		wscript.echo "TAGUPDATE: KK.0 = True"
	else
		wscript.echo "TAGUPDATE: KK.0 = False"
	end if

	

end sub


'
sub forceCscript
set shell = createObject("wscript.shell")
	If Instr(1, WScript.FullName, "CScript", vbTextCompare) = 0 Then
		shell.Run "cscript """ & WScript.ScriptFullName & """", 1, False
		WScript.Quit
	End If
end sub