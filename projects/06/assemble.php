#!/usr/bin/php
<?php

function err($msg) {
  echo "Error: {$msg}\n";
  exit(1);
}

function err_syntax($msg) {
  echo "Syntax Error: {$msg}\n";
  exit(1);
}

if ($argc != 2) {
  err("Usage: {$argv[0]} <file>");
}

$fileName = $argv[1];
$data = file($fileName, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
if ($data === false) {
  err("Unable to read {$argv[1]}");
}

$compLookup = [
  "0" => "0101010",
  "1" => "0111111",
  "-1" => "0111010",
  "D" => "0001100",
  "A" => "0110000",
  "M" => "1110000",
  "!D" => "0001101",
  "!A" => "0110001",
  "!M" => "1110001",
  "-D" => "0001111",
  "-A" => "0110011",
  "-M" => "1110011",
  "D+1" => "0011111",
  "A+1" => "0110111",
  "M+1" => "1110111",
  "D-1" => "0001110",
  "A-1" => "0110010",
  "M-1" => "1110010",
  "D+A" => "0000010",
  "D+M" => "1000010",
  "D-A" => "0010011",
  "D-M" => "1010011",
  "A-D" => "0000111",
  "M-D" => "1000111",
  "D&A" => "0000000",
  "D&M" => "1000000",
  "D|A" => "0010101",
  "D|M" => "1010101"
];

$destLookup = [
  "M" => "001",
  "D" => "010",
  "MD" => "011",
  "A" => "100",
  "AM" => "101",
  "AD" => "110",
  "AMD" => "111"
];

$jumpLookup = [
  "JGT" => "001",
  "JEQ" => "010",
  "JGE" => "011",
  "JLT" => "100",
  "JNE" => "101",
  "JLE" => "110",
  "JMP" => "111"
];

$predefinedLookup = [
  "R0" => 0,
  "R1" => 1,
  "R2" => 2,
  "R3" => 3,
  "R4" => 4,
  "R5" => 5,
  "R6" => 6,
  "R7" => 7,
  "R8" => 8,
  "R9" => 9,
  "R10" => 10,
  "R11" => 11,
  "R12" => 12,
  "R13" => 13,
  "R14" => 14,
  "R15" => 15,
  "SCREEN" => 16384,
  "KBD" => 24576,
  "SP" => 0,
  "LCL" => 1,
  "ARG" => 2,
  "THIS" => 3,
  "THAT" => 4
];

$processedData = [];
$lineNo = 0;
foreach ($data as $line) {
  // remove comments
  $line = trim($line);
  if (preg_match("/\/\/.*$/", $line) === 1) {
    $line = preg_replace("/\/\/.*$/", "", $line);
  }

  $line = trim($line);
  if (strlen($line) == 0) {
    continue;
  }

  if (preg_match("/^\((.*)\)$/", $line, $matches) === 1) {
    $name = $matches[1];
    $predefinedLookup[$name] = $lineNo;

    continue;
  }

  array_push($processedData, $line);
  // printf("%d: %s\n", $lineNo, $line);

  $lineNo++;
}

$variableLocation = 16;

$assembledCode = [];
foreach ($processedData as $line) {
  if (preg_match("/^@([0-9]*)$/", $line, $matches) === 1) {
    $number = intval($matches[1]);
    $code = sprintf("0%015b\n", $number);
    array_push($assembledCode, $code);

    continue;
  }

  if (preg_match("/^@([A-Za-z0-9$._]*)$/", $line, $matches) === 1) {
    $name = $matches[1];
    if (!array_key_exists($name, $predefinedLookup)) {
      $predefinedLookup[$name] = $variableLocation;
      $variableLocation++;
    }

    $location = $predefinedLookup[$name];
    $code = sprintf("0%015b\n", $location);
    array_push($assembledCode, $code);

    continue;
  }

  // dest=comp;jump
  $line = str_replace(' ', '', $line);

  $jumpArr = explode(';', $line);
  if (count($jumpArr) == 2) {
    $jumpSection = $jumpArr[1];
    if (array_key_exists($jumpSection, $jumpLookup)) {
      $jumpCode = $jumpLookup[$jumpSection];
    } else {
      err_syntax("No jump code [{$jumpSection}] in [{$lineNo}][${line}]");
    }
  } else {
    $jumpCode = "000";
  }

  $compArr = explode('=', $jumpArr[0]);
  if (count($compArr) == 2) {
    $destSection = $compArr[0];
    if (array_key_exists($destSection, $destLookup)) {
      $destCode = $destLookup[$destSection];
    } else {
      err_syntax("No dest code [{$destSection}] in [{$lineNo}][${line}]");
    }

    $compSection = $compArr[1];
  } else {
    $destCode = "000";

    $compSection = $compArr[0];
  }

  if (array_key_exists($compSection, $compLookup)) {
    $compCode = $compLookup[$compSection];
  } else {
    err_syntax("No comp code [{$compSection}] in [{$lineNo}][${line}]");
  }

  array_push($assembledCode, "111{$compCode}{$destCode}{$jumpCode}\n");
}

$outputName = dirname($fileName) . "/" . basename($fileName, ".asm") . ".hack";
file_put_contents($outputName, $assembledCode);
// var_dump(json_encode($predefinedLookup, JSON_PRETTY_PRINT));
