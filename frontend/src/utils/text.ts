export function initials(name){
return name.split(' ').map(s=>s[0]).join('').slice(0,2).toUpperCase()
}
export function pickColor(seed){
const colors=['#6A9C89','#4C7A6F','#6B7280','#A78BFA','#F59E0B','#E11D48','#22D3EE']
const n=[...seed].reduce((a,c)=>a+c.charCodeAt(0),0)
return colors[n%colors.length]
}