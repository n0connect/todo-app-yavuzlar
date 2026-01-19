//go:build cgo

package cryptoengine

/*
#cgo pkg-config: openssl
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/kdf.h>
#include <openssl/core_names.h>
#include <openssl/params.h>
#include <openssl/rand.h>
#include <openssl/err.h>
#include <openssl/crypto.h>
#include <stdlib.h>
#include <string.h>

static int argon2id_derive(const unsigned char *password, size_t password_len,
	const unsigned char *salt, size_t salt_len,
	uint32_t iter, uint32_t mem, uint32_t lanes,
	uint32_t out_len, unsigned char *out) {
	EVP_KDF *kdf = EVP_KDF_fetch(NULL, "ARGON2ID", NULL);
	if (!kdf) {
		return 0;
	}
	EVP_KDF_CTX *ctx = EVP_KDF_CTX_new(kdf);
	EVP_KDF_free(kdf);
	if (!ctx) {
		return 0;
	}
	uint32_t version = 0x13;
	OSSL_PARAM params[] = {
		OSSL_PARAM_construct_octet_string(OSSL_KDF_PARAM_PASSWORD, (void *)password, password_len),
		OSSL_PARAM_construct_octet_string(OSSL_KDF_PARAM_SALT, (void *)salt, salt_len),
		OSSL_PARAM_construct_uint32(OSSL_KDF_PARAM_ITER, &iter),
		OSSL_PARAM_construct_uint32(OSSL_KDF_PARAM_ARGON2_MEMCOST, &mem),
		OSSL_PARAM_construct_uint32(OSSL_KDF_PARAM_ARGON2_LANES, &lanes),
		OSSL_PARAM_construct_uint32(OSSL_KDF_PARAM_ARGON2_VERSION, &version),
		OSSL_PARAM_construct_uint32(OSSL_KDF_PARAM_SIZE, &out_len),
		OSSL_PARAM_construct_end()
	};
	int ok = EVP_KDF_derive(ctx, out, out_len, params);
	EVP_KDF_CTX_free(ctx);
	return ok;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

func opensslError(context string) error {
	errCode := C.ERR_get_error()
	if errCode == 0 {
		return fmt.Errorf("%s: openssl error", context)
	}
	buf := make([]byte, 256)
	C.ERR_error_string_n(errCode, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	return fmt.Errorf("%s: %s", context, bytesToString(buf))
}

func bytesToString(b []byte) string {
	for i, v := range b {
		if v == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func RandomBytes(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size")
	}
	out := make([]byte, size)
	if C.RAND_bytes((*C.uchar)(unsafe.Pointer(&out[0])), C.int(size)) != 1 {
		return nil, opensslError("RAND_bytes")
	}
	return out, nil
}

func SHA256(data []byte) ([]byte, error) {
	ctx := C.EVP_MD_CTX_new()
	if ctx == nil {
		return nil, errors.New("EVP_MD_CTX_new failed")
	}
	defer C.EVP_MD_CTX_free(ctx)

	if C.EVP_DigestInit_ex(ctx, C.EVP_sha256(), nil) != 1 {
		return nil, opensslError("EVP_DigestInit_ex")
	}

	if len(data) > 0 {
		if C.EVP_DigestUpdate(ctx, unsafe.Pointer(&data[0]), C.size_t(len(data))) != 1 {
			return nil, opensslError("EVP_DigestUpdate")
		}
	}

	out := make([]byte, SHA256Size)
	var outLen C.uint
	if C.EVP_DigestFinal_ex(ctx, (*C.uchar)(unsafe.Pointer(&out[0])), &outLen) != 1 {
		return nil, opensslError("EVP_DigestFinal_ex")
	}
	if int(outLen) != SHA256Size {
		return nil, fmt.Errorf("unexpected digest length: %d", outLen)
	}
	return out, nil
}

func HMACSHA256(key, data []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("empty key")
	}
	out := make([]byte, SHA256Size)
	var outLen C.uint
	var keyPtr unsafe.Pointer
	if len(key) > 0 {
		keyPtr = unsafe.Pointer(&key[0])
	}
	var dataPtr *C.uchar
	if len(data) > 0 {
		dataPtr = (*C.uchar)(unsafe.Pointer(&data[0]))
	}

	res := C.HMAC(
		C.EVP_sha256(),
		keyPtr,
		C.int(len(key)),
		dataPtr,
		C.size_t(len(data)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		&outLen,
	)
	if res == nil {
		return nil, opensslError("HMAC")
	}
	if int(outLen) != SHA256Size {
		return nil, fmt.Errorf("unexpected hmac length: %d", outLen)
	}
	return out, nil
}

func ConstantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return C.CRYPTO_memcmp(unsafe.Pointer(&a[0]), unsafe.Pointer(&b[0]), C.size_t(len(a))) == 0
}

func Argon2idKey(password, salt []byte, iterations, memory, lanes, keyLen uint32) ([]byte, error) {
	if len(password) == 0 || len(salt) == 0 {
		return nil, fmt.Errorf("invalid input")
	}
	if keyLen == 0 {
		return nil, fmt.Errorf("invalid key length")
	}

	out := make([]byte, keyLen)
	if C.argon2id_derive(
		(*C.uchar)(unsafe.Pointer(&password[0])),
		C.size_t(len(password)),
		(*C.uchar)(unsafe.Pointer(&salt[0])),
		C.size_t(len(salt)),
		C.uint32_t(iterations),
		C.uint32_t(memory),
		C.uint32_t(lanes),
		C.uint32_t(keyLen),
		(*C.uchar)(unsafe.Pointer(&out[0])),
	) != 1 {
		return nil, opensslError("argon2id_derive")
	}
	return out, nil
}

func EncryptAES256GCM(key, plaintext, aad []byte) ([]byte, error) {
	if len(key) != AES256KeySize {
		return nil, fmt.Errorf("invalid key length")
	}

	nonce, err := RandomBytes(GCMNonceSize)
	if err != nil {
		return nil, err
	}

	ctx := C.EVP_CIPHER_CTX_new()
	if ctx == nil {
		return nil, errors.New("EVP_CIPHER_CTX_new failed")
	}
	defer C.EVP_CIPHER_CTX_free(ctx)

	if C.EVP_EncryptInit_ex(ctx, C.EVP_aes_256_gcm(), nil, nil, nil) != 1 {
		return nil, opensslError("EVP_EncryptInit_ex")
	}
	if C.EVP_CIPHER_CTX_ctrl(ctx, C.EVP_CTRL_GCM_SET_IVLEN, C.int(len(nonce)), nil) != 1 {
		return nil, opensslError("EVP_CTRL_GCM_SET_IVLEN")
	}
	if C.EVP_EncryptInit_ex(ctx, nil, nil, (*C.uchar)(unsafe.Pointer(&key[0])), (*C.uchar)(unsafe.Pointer(&nonce[0]))) != 1 {
		return nil, opensslError("EVP_EncryptInit_ex(key)")
	}

	var outLen C.int
	if len(aad) > 0 {
		if C.EVP_EncryptUpdate(ctx, nil, &outLen, (*C.uchar)(unsafe.Pointer(&aad[0])), C.int(len(aad))) != 1 {
			return nil, opensslError("EVP_EncryptUpdate(AAD)")
		}
	}

	ciphertext := make([]byte, len(plaintext)+1)
	if len(plaintext) > 0 {
		if C.EVP_EncryptUpdate(ctx, (*C.uchar)(unsafe.Pointer(&ciphertext[0])), &outLen, (*C.uchar)(unsafe.Pointer(&plaintext[0])), C.int(len(plaintext))) != 1 {
			return nil, opensslError("EVP_EncryptUpdate")
		}
	} else {
		outLen = 0
	}
	ciphertextLen := int(outLen)

	if C.EVP_EncryptFinal_ex(ctx, (*C.uchar)(unsafe.Pointer(&ciphertext[ciphertextLen])), &outLen) != 1 {
		return nil, opensslError("EVP_EncryptFinal_ex")
	}
	ciphertextLen += int(outLen)
	ciphertext = ciphertext[:ciphertextLen]

	tag := make([]byte, GCMTagSize)
	if C.EVP_CIPHER_CTX_ctrl(ctx, C.EVP_CTRL_GCM_GET_TAG, C.int(len(tag)), unsafe.Pointer(&tag[0])) != 1 {
		return nil, opensslError("EVP_CTRL_GCM_GET_TAG")
	}

	out := make([]byte, 0, len(nonce)+len(ciphertext)+len(tag))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	out = append(out, tag...)
	return out, nil
}

func DecryptAES256GCM(key, data, aad []byte) ([]byte, error) {
	if len(key) != AES256KeySize {
		return nil, fmt.Errorf("invalid key length")
	}
	if len(data) < GCMNonceSize+GCMTagSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := data[:GCMNonceSize]
	tag := data[len(data)-GCMTagSize:]
	ciphertext := data[GCMNonceSize : len(data)-GCMTagSize]

	ctx := C.EVP_CIPHER_CTX_new()
	if ctx == nil {
		return nil, errors.New("EVP_CIPHER_CTX_new failed")
	}
	defer C.EVP_CIPHER_CTX_free(ctx)

	if C.EVP_DecryptInit_ex(ctx, C.EVP_aes_256_gcm(), nil, nil, nil) != 1 {
		return nil, opensslError("EVP_DecryptInit_ex")
	}
	if C.EVP_CIPHER_CTX_ctrl(ctx, C.EVP_CTRL_GCM_SET_IVLEN, C.int(len(nonce)), nil) != 1 {
		return nil, opensslError("EVP_CTRL_GCM_SET_IVLEN")
	}
	if C.EVP_DecryptInit_ex(ctx, nil, nil, (*C.uchar)(unsafe.Pointer(&key[0])), (*C.uchar)(unsafe.Pointer(&nonce[0]))) != 1 {
		return nil, opensslError("EVP_DecryptInit_ex(key)")
	}

	var outLen C.int
	if len(aad) > 0 {
		if C.EVP_DecryptUpdate(ctx, nil, &outLen, (*C.uchar)(unsafe.Pointer(&aad[0])), C.int(len(aad))) != 1 {
			return nil, opensslError("EVP_DecryptUpdate(AAD)")
		}
	}

	plaintext := make([]byte, len(ciphertext)+1)
	if len(ciphertext) > 0 {
		if C.EVP_DecryptUpdate(ctx, (*C.uchar)(unsafe.Pointer(&plaintext[0])), &outLen, (*C.uchar)(unsafe.Pointer(&ciphertext[0])), C.int(len(ciphertext))) != 1 {
			return nil, opensslError("EVP_DecryptUpdate")
		}
	} else {
		outLen = 0
	}
	plainLen := int(outLen)

	if C.EVP_CIPHER_CTX_ctrl(ctx, C.EVP_CTRL_GCM_SET_TAG, C.int(len(tag)), unsafe.Pointer(&tag[0])) != 1 {
		return nil, opensslError("EVP_CTRL_GCM_SET_TAG")
	}

	if C.EVP_DecryptFinal_ex(ctx, (*C.uchar)(unsafe.Pointer(&plaintext[plainLen])), &outLen) != 1 {
		return nil, fmt.Errorf("decryption failed")
	}
	plainLen += int(outLen)
	return plaintext[:plainLen], nil
}
