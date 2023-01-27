package email

const paymentNotify = `
{{define "paymentNotify"}}
<div>
	<h1>{{$.title}}</h1>
	<p>Hi {{$.name}}. You received this email because {{$.name}} from 
		<a target="_blank" href="{{$.link}}">mgmt-ng</a> sent you a payment request.</p>
	<p>Please click on <a target="_blank" href="{{$.link}}">here</a> to see the detail</p>
</div>
{{end}}
`
