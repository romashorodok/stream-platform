
// const TOKEN = "eyJhbGciOiJFUzI1NiIsImtpZCI6ImZha2Uta2V5LWlkIiwidHlwIjoiSldUIn0.eyJhdWQiOlsiZXhhbXBsZS11c2VycyJdLCJpc3MiOiJmYWtlLWlzc3VlciIsInBlcm0iOlsidGhpbmdzOnciXX0.CbPT1hzWmyTt0lTyv-fiyUlnY1SGa0vrX52yFjeigx2PA1-78LVH0z5hukPKkLMPDMXL9AJrtNp0elWSD_qrBw";

export async function load({ fetch }: { fetch: typeof window.fetch }) {

	// const result = fetch("http://localhost:8082/stream:stat", {
	// });

	// console.log(await result)

	// console.log("I'm request")

	return {
		posts: {
			one: "tets",
			two: "test1",
		}
	};
}
