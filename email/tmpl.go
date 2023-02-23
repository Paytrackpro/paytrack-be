package email

const paymentNotify = `
{{define "paymentNotify"}}
<div>
	<h1>{{$.Title}}</h1>
	<p>Hi {{$.Receiver}}. You received this email because {{$.Sender}} from 
		<a target="_blank" href="{{$.Link}}">mgmt-ng</a> sent you a payment {{if $.IsRequest}}request{{else}}reminder{{end}}.</p>
	<p>Please click on <a target="_blank" href="{{$.Link}}{{$.Path}}">here</a> to see the detail</p>
</div>
{{end}}
`
