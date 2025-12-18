package utils

import (
	"fmt"

	"github.com/Agushim/go_wifi_billing/models"
)

func BuildBillMessage(bill models.Bill) string {
	return fmt.Sprintf(
		"Assalamualaikum,\n\nBapak/Ibu %s,\n\n"+
			"Kami informasikan bahwa tagihan layanan internet untuk bulan %s\n\n"+
			"🧾 Nomor Tagihan: %s\n"+
			"💰 Total Tagihan: Rp %s\n"+
			"📅 Jatuh Tempo: %s\n\n"+
			"Silakan melakukan pembayaran melalui link berikut: %s\n\n"+
			"Terima kasih atas kepercayaan Bapak/Ibu menggunakan layanan kami.\n"+
			"Jika ada pertanyaan, silakan hubungi kami.\n\nHormat kami,\nPT Cantika Data Prima",
		bill.Customer.User.Name,
		bill.BillDate.Format("January"),
		bill.PublicID,
		FormatCurrency(bill.Amount),
		bill.DueDate.Format("02 January 2006"),
		fmt.Sprintf("https://cantika.net/%s", bill.PublicID),
	)
}

func BuildReminderMessage(bill models.Bill) string {
	return fmt.Sprintf(
		"Assalamualaikum, Bapak/Ibu %s,\n\n"+
			"Kami mengingatkan bahwa tagihan layanan internet Anda akan *jatuh tempo dalam 5 hari*.\n\n"+
			"🧾 Nomor Tagihan: %s\n"+
			"💰 Total Tagihan: Rp %s\n"+
			"📅 Jatuh Tempo: %s\n\n"+
			"Silakan melakukan pembayaran melalui link berikut: %s\n\n"+
			"Terima kasih atas kepercayaan Bapak/Ibu menggunakan layanan kami.\n"+
			"Jika ada pertanyaan, silakan hubungi kami.\n\nHormat kami,\nPT Cantika Data Prima",
		bill.Customer.User.Name,
		bill.PublicID,
		FormatCurrency(bill.Amount),
		bill.DueDate.Format("02 January 2006"),
		fmt.Sprintf("https://cantika.net/%s", bill.PublicID),
	)
}
