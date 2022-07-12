var professori = undefined;

$("#corsoDiLaurea").on('change', (event) => {
	$.ajax({
		url: "/FantaUniBo/getprofessori.jsp?id=" + $("#corsoDiLaurea").val(),
		success: (data) => {
			professori = data;
				
			$(".professori").each((index, element) => {
				$(element).html("");
				
				professori.forEach((professore) => {
					$(element).append(`<option value="${professore.id}">${professore.nome} ${professore.cognome}</option>`);
				})
			})
		}
	})
});

$("#buttonAdd").on('click', (event) => {
	var clone = $("#professoreSeguito1").clone();
	
	clone.attr("id", "professoreSeguito" + ($(".professoreSeguito").length + 1));
	
	//clone.find(".professoreSeguito").attr("name", "professoreSeguito"+($(".professoreSeguito").length + 1));
	clone.find("button").attr("target", clone.attr("id")).prop("disabled", false).on('click', (event) => {
		$("#" + $(event.target).attr("target")).remove();
	});
	
	clone.appendTo("#professoreSeguiti");
})