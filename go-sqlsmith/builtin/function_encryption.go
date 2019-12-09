package builtin


import "github.com/pingcap/parser/ast"

// compress and uncompress function make diff
var encryptionFunctions = []*functionClass{
	{ast.AesDecrypt, 2, 3, false, true, false},
	{ast.AesEncrypt, 2, 3, false, true, false},
	// {ast.Compress, 1, 1, false, true, false},
	// removed in MySQL 8.0.3
	// {ast.Decode, 2, 2, false, true, false},
	{ast.DesDecrypt, 1, 2, false, true, false},
	{ast.DesEncrypt, 1, 2, false, true, false},
	// {ast.Encode, 2, 2, false, true, false},
	{ast.Encrypt, 1, 2, false, true, false},
	{ast.MD5, 1, 1, false, true, false},
	{ast.OldPassword, 1, 1, false, true, false},
	// {ast.PasswordFunc, 1, 1, false, true, false},
	{ast.RandomBytes, 1, 1, false, true, false},
	{ast.SHA1, 1, 1, false, true, false},
	{ast.SHA, 1, 1, false, true, false},
	{ast.SHA2, 2, 2, false, true, false},
	// {ast.Uncompress, 1, 1, false, true, false},
	// {ast.UncompressedLength, 1, 1, false, true, false},
	// {ast.ValidatePasswordStrength, 1, 1, false, true, false},
}
